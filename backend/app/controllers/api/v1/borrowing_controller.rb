class Api::V1::BorrowingController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :set_borrow_record, only: [:show, :approve, :reject]

  # GET /api/v1/borrowing/requests
  def index
    member = current_user.member
    raise "Active member account required" unless member && member.status == "active"

    scope = BorrowRecord
      .where(member_id: member.id)
      .order(created_at: :desc)

    if params[:status].present?
      scope = scope.where(status: params[:status])
    end

    if params[:pending].present? && params[:pending] == "true"
      scope = scope.where(status: "pending").or(scope.where(status: "approved"))
    end

    records = paginate(scope.includes(:inventory_item, :approved_by))

    render json: {
      success: true,
      requests: records.items.map { |r| serialize_borrow_record(r) },
      pagination: {
        page: records.current_page,
        per_page: records.per_page,
        total_count: records.total_count
      }
    }, status: :ok
  end

  # GET /api/v1/borrowing/requests/:id
  def show
    render json: {
      success: true,
      request: serialize_borrow_record_full(@borrow_record)
    }, status: :ok
  end

  # POST /api/v1/borrowing/requests
  def create
    member = current_user.member
    raise "Active member account required" unless member && member.status == "active"

    item_id = params[:item_id].to_i
    quantity = params[:quantity].to_i
    purpose = params[:purpose].to_s.strip

    unless item_id.present?
      return render json: {
        success: false,
        message: "Item ID is required"
      }, status: :unprocessable_entity
    end

    unless quantity.present? && quantity > 0
      return render json: {
        success: false,
        message: "Quantity must be greater than zero"
      }, status: :unprocessable_entity
    end

    inventory_item = InventoryItem.find_by(id: item_id, is_active: true, deleted_at: nil)
    unless inventory_item
      return render json: {
        success: false,
        message: "Inventory item not found"
      }, status: :not_found
    end

    unless inventory_item.available_quantity >= quantity
      return render json: {
        success: false,
        message: "Insufficient quantity available (requested: #{quantity}, available: #{inventory_item.available_quantity})"
      }, status: :unprocessable_entity
    end

    existing_borrow = BorrowRecord
      .where(member_id: member.id, inventory_item_id: item_id)
      .where.not(status: "returned")
      .first

    if existing_borrow
      return render json: {
        success: false,
        message: "You already have an active borrow record for this item"
      }, status: :conflict
    end

    due_date = Semester.current&.end_date || 14.days.from_now.to_date

    idempotency_key = SecureRandom.uuid

    begin
      borrow_record = BorrowRecord.create!(
        member_id: member.id,
        inventory_item_id: item_id,
        quantity: quantity,
        status: "pending",
        borrowed_at: Time.current,
        due_date: due_date,
        idempotency_key: idempotency_key,
        approval_reason: purpose.presence
      )

      InventoryItem.decrement_counter(:available_quantity, item_id, by: quantity)

      AuditLogService.log_borrow_request(borrow_record, requested_by: current_user)

      NotificationService.notify_borrow_request(borrow_record, requested_by: current_user)

      render json: {
        success: true,
        message: "Borrow request submitted successfully",
        request: serialize_borrow_record(borrow_record)
      }, status: :created
    rescue ActiveRecord::RecordInvalid => e
      InventoryItem.increment_counter(:available_quantity, item_id, by: quantity)
      render json: {
        success: false,
        message: e.message
      }, status: :unprocessable_entity
    rescue => e
      Rails.logger.error("Borrow request creation failed: #{e.class} - #{e.message}")
      render json: {
        success: false,
        message: "Failed to submit borrow request"
      }, status: :internal_server_error
    end
  end

  # POST /api/v1/borrowing/requests/:id/approve
  def approve
    raise "Administrator access required" unless current_user.admin?

    unless @borrow_record.status == "pending"
      return render json: {
        success: false,
        message: "Only pending requests can be approved"
      }, status: :unprocessable_entity
    end

    ActiveRecord::Base.transaction do
      @borrow_record.update!(
        status: "approved",
        approved_by_id: current_user.id,
        approval_reason: params[:approval_reason].to_s.presence
      )
    end

    AuditLogService.log_borrow_approved(
      @borrow_record,
      approved_by: current_user,
      ip_address: request.remote_ip
    )

    NotificationService.notify_borrow_approved(@borrow_record, approved_by: current_user)

    render json: {
      success: true,
      message: "Borrow request approved",
      request: serialize_borrow_record_full(@borrow_record)
    }, status: :ok
  end

  # POST /api/v1/borrowing/requests/:id/reject
  def reject
    raise "Administrator access required" unless current_user.admin?

    unless @borrow_record.status == "pending"
      return render json: {
        success: false,
        message: "Only pending requests can be rejected"
      }, status: :unprocessable_entity
    end

    reason = params[:reason].to_s.strip
    unless reason.present?
      return render json: {
        success: false,
        message: "Rejection reason is required"
      }, status: :unprocessable_entity
    end

    ActiveRecord::Base.transaction do
      @borrow_record.update!(
        status: "rejected",
        approval_reason: reason
      )
      item = @borrow_record.inventory_item
      restored = [item.available_quantity + @borrow_record.quantity, item.total_quantity].min
      item.update!(available_quantity: restored)
    end

    AuditLogService.log_borrow_rejected(
      @borrow_record,
      rejected_by: current_user,
      reason: reason,
      ip_address: request.remote_ip
    )

    NotificationService.notify_borrow_rejected(@borrow_record, rejected_by: current_user, reason: reason)

    render json: {
      success: true,
      message: "Borrow request rejected",
      request: serialize_borrow_record_full(@borrow_record)
    }, status: :ok
  end

  # POST /api/v1/borrowing/extension-request
  def request_extension
    member = current_user.member
    raise "Active member account required" unless member && member.status == "active"

    borrow_record_id = params[:borrow_record_id].to_i
    reason = params[:reason].to_s.strip

    unless borrow_record_id.positive?
      return render json: {
        success: false,
        message: "Borrow record ID is required"
      }, status: :unprocessable_entity
    end

    borrow_record = BorrowRecord.includes(:inventory_item).find_by(
      id: borrow_record_id,
      member_id: member.id,
      status: "approved"
    )

    unless borrow_record
      return render json: {
        success: false,
        message: "Active borrow record not found"
      }, status: :not_found
    end

    unless borrow_record.due_date < Date.current
      return render json: {
        success: false,
        message: "Extension can only be requested for overdue items"
      }, status: :unprocessable_entity
    end

    if reason.blank?
      return render json: {
        success: false,
        message: "Reason is required"
      }, status: :unprocessable_entity
    end

    requested_return_date =
      if params[:requested_return_date].present?
        begin
          Date.parse(params[:requested_return_date].to_s)
        rescue Date::Error
          return render json: {
            success: false,
            message: "Requested return date is invalid"
          }, status: :unprocessable_entity
        end
      else
        Date.current + 14.days
      end

    if requested_return_date <= borrow_record.due_date
      return render json: {
        success: false,
        message: "Requested return date must be after the current due date"
      }, status: :unprocessable_entity
    end

    extension = borrow_record.extension_request || ExtensionRequest.create!(
      borrow_record_id: borrow_record.id,
      member_id: member.id,
      requested_return_date: requested_return_date,
      reason: reason,
      status: "pending"
    )

    AuditLogService.log!(
      user: current_user,
      action: "extension_requested",
      resource_type: "ExtensionRequest",
      resource_id: extension.id,
      details: {
        item_name: borrow_record.inventory_item.name,
        current_due_date: borrow_record.due_date.to_s,
        requested_return_date: extension.requested_return_date.to_s,
        reason: reason
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Extension request submitted",
      extension: serialize_extension_request(extension)
    }, status: :created
  end

  # GET /api/v1/borrowing/admin/requests
  def admin_index
    raise "Administrator access required" unless current_user.admin?

    scope = BorrowRecord
      .includes(:member, :inventory_item)
      .order(created_at: :desc)

    if params[:status].present?
      scope = scope.where(status: params[:status])
    end

    if params[:pending].present? && params[:pending] == "true"
      scope = scope.where(status: "pending")
    end

    if params[:overdue].present? && params[:overdue] == "true"
      scope = scope.where("due_date < ? AND status IN ('approved', 'borrowed')", Date.current)
    end

    if params[:member_id].present?
      scope = scope.where(member_id: params[:member_id])
    end

    if params[:inventory_item_id].present?
      scope = scope.where(inventory_item_id: params[:inventory_item_id])
    end

    records = paginate(scope)

    render json: {
      success: true,
      requests: records.items.map { |r| serialize_borrow_record_admin(r) },
      pagination: {
        page: records.current_page,
        per_page: records.per_page,
        total_count: records.total_count,
        pending_count: BorrowRecord.where(status: "pending").count,
        overdue_count: BorrowRecord.where(status: "approved", due_date: ..Date.current).count
      }
    }, status: :ok
  end

  private

  def set_borrow_record
    @borrow_record = BorrowRecord.includes(:member, :inventory_item, :approved_by, :extension_request)
      .find(params[:id])
  rescue ActiveRecord::RecordNotFound
    render json: {
      success: false,
      message: "Borrow record not found"
    }, status: :not_found
  end


  def serialize_borrow_record(borrow)
    {
      id: borrow.id,
      item: {
        id: borrow.inventory_item.id,
        name: borrow.inventory_item.name,
        category: borrow.inventory_item.category
      },
      quantity: borrow.quantity,
      status: borrow.status,
      borrowed_at: borrow.borrowed_at,
      due_date: borrow.due_date.to_s,
      returned_at: borrow.returned_at
    }
  end

  def serialize_borrow_record_full(borrow)
    serialize_borrow_record(borrow).merge(
      purpose: borrow.approval_reason,
      approved_by: borrow.approved_by&.email,
      approval_reason: borrow.approval_reason,
      extension_request: borrow.extension_request && serialize_extension_request(borrow.extension_request)
    )
  end

  def serialize_borrow_record_admin(borrow)
    {
      id: borrow.id,
      member: {
        id: borrow.member.id,
        name: borrow.member.name,
        login_id: borrow.member.login_id,
        email: borrow.member.email
      },
      item: {
        id: borrow.inventory_item.id,
        name: borrow.inventory_item.name,
        category: borrow.inventory_item.category
      },
      quantity: borrow.quantity,
      status: borrow.status,
      borrowed_at: borrow.borrowed_at,
      due_date: borrow.due_date.to_s,
      days_overdue: borrow.due_date < Date.current ? (Date.current - borrow.due_date).to_i : 0,
      returned_at: borrow.returned_at,
      approved_by: borrow.approved_by&.email,
      approval_reason: borrow.approval_reason
    }
  end

  def serialize_extension_request(extension)
    {
      id: extension.id,
      borrow_record_id: extension.borrow_record_id,
      status: extension.status,
      requested_return_date: extension.requested_return_date.to_s,
      reason: extension.reason,
      created_at: extension.created_at
    }
  end
end
