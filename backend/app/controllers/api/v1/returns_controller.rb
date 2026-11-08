class Api::V1::ReturnsController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :set_return_request, only: [:show, :confirm_return]
  before_action :set_borrow_record_for_return, only: [:create, :request_return]

  # GET /api/v1/returns/requests
  def index
    member = current_user.member
    raise "Active member account required" unless member && member.status == "active"

    scope = ReturnRequest
      .where(member_id: member.id)
      .order(created_at: :desc)

    requests = paginate(scope.includes(:borrow_record))

    render json: {
      success: true,
      requests: requests.map { |r| serialize_return_request(r) },
      pagination: {
        page: requests.current_page,
        per_page: requests.per_page,
        total_count: requests.total_count
      }
    }, status: :ok
  end

  # POST /api/v1/returns/request
  def create
    member = current_user.member
    raise "Active member account required" unless member && member.status == "active"

    borrow_record = BorrowRecord
      .includes(:inventory_item, :member)
      .find_by(id: params[:borrow_record_id], member_id: member.id)

    unless borrow_record
      return render json: {
        success: false,
        message: "Borrow record not found"
      }, status: :not_found
    end

    unless %w[approved borrowed].include?(borrow_record.status)
      return render json: {
        success: false,
        message: "Cannot request return for this borrow status"
      }, status: :unprocessable_entity
    end

    existing_return = ReturnRequest.find_by(borrow_record_id: borrow_record.id)
    if existing_return
      return render json: {
        success: false,
        message: "Return request already exists for this borrow"
      }, status: :conflict
    end

    ActiveRecord::Base.transaction do
      return_request = ReturnRequest.create!(
        borrow_record_id: borrow_record.id,
        member_id: member.id,
        status: "pending",
        condition_note: params[:condition_note].to_s.strip.presence
      )

      AuditLogService.log!(
        user: current_user,
        action: "return_requested",
        resource_type: "ReturnRequest",
        resource_id: return_request.id,
        details: {
          item_name: borrow_record.inventory_item.name,
          item_category: borrow_record.inventory_item.category,
          borrow_date: borrow_record.borrowed_at.to_s,
          due_date: borrow_record.due_date.to_s
        },
        ip_address: request.remote_ip
      )
    end

    render json: {
      success: true,
      message: "Return request submitted",
      request: serialize_return_request(ReturnRequest.last)
    }, status: :created
  end

  # GET /api/v1/returns/admin/requests
  def admin_index
    raise "Administrator access required" unless current_user.admin?

    scope = ReturnRequest
      .includes(:borrow_record, :member, :confirmed_by)
      .order(created_at: :desc)

    if params[:status].present?
      scope = scope.where(status: params[:status])
    end

    if params[:pending].present? && params[:pending] == "true"
      scope = scope.where(status: "pending")
    end

    if params[:member_id].present?
      scope = scope.where(member_id: params[:member_id])
    end

    requests = paginate(scope)

    render json: {
      success: true,
      requests: requests.map { |r| serialize_return_request_admin(r) },
      pagination: {
        page: requests.current_page,
        per_page: requests.per_page,
        total_count: requests.total_count,
        pending_count: ReturnRequest.where(status: "pending").count
      }
    }, status: :ok
  end

  # POST /api/v1/returns/:id/confirm
  def confirm_return
    raise "Administrator access required" unless current_user.admin?

    return_request = ReturnRequest.find(params[:id])

    unless return_request.status == "pending"
      return render json: {
        success: false,
        message: "Only pending return requests can be confirmed"
      }, status: :unprocessable_entity
    end

    ActiveRecord::Base.transaction do
      borrow_record = return_request.borrow_record
      inventory_item = borrow_record.inventory_item

      borrow_record.update!(
        status: "returned",
        returned_at: Time.current,
        condition_on_return: params[:condition_note].to_s.strip.presence
      )

      return_request.update!(
        status: "confirmed",
        confirmed_by_id: current_user.id,
        confirmed_at: Time.current,
        condition_note: params[:condition_note].to_s.strip.presence
      )

      inventory_item.update!(
        available_quantity: [
          inventory_item.available_quantity + borrow_record.quantity,
          inventory_item.total_quantity
        ].min
      )
    end

    AuditLogService.log_return_confirmed(
      return_request,
      confirmed_by: current_user,
      ip_address: request.remote_ip
    )

    NotificationService.notify_return_confirmed(return_request, confirmed_by: current_user)

    render json: {
      success: true,
      message: "Return confirmed. Item is now available in inventory",
      request: serialize_return_request_full(return_request)
    }, status: :ok
  end

  # POST /api/v1/returns/:id/reject
  def reject
    raise "Administrator access required" unless current_user.admin?

    return_request = ReturnRequest.find(params[:id])

    unless return_request.status == "pending"
      return render json: {
        success: false,
        message: "Only pending return requests can be rejected"
      }, status: :unprocessable_entity
    end

    reason = params[:reason].to_s.strip
    unless reason.present?
      return render json: {
        success: false,
        message: "Rejection reason is required"
      }, status: :unprocessable_entity
    end

    return_request.update!(
      status: "rejected",
      condition_note: reason
    )

    AuditLogService.log!(
      user: current_user,
      action: "return_request_rejected",
      resource_type: "ReturnRequest",
      resource_id: return_request.id,
      details: {
        member_login_id: return_request.member.login_id,
        item_name: return_request.borrow_record.inventory_item.name,
        reason: reason
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Return request rejected",
      request: serialize_return_request(return_request)
    }, status: :ok
  end

  private

  def set_return_request
    @return_request = ReturnRequest
      .includes(:borrow_record, :member, :confirmed_by, :borrow_record => [:inventory_item])
      .find(params[:id])
  rescue ActiveRecord::RecordNotFound
    render json: {
      success: false,
      message: "Return request not found"
    }, status: :not_found
  end

  def set_borrow_record_for_return
    @borrow_record = BorrowRecord.includes(:inventory_item, :member)
      .find_by(id: params[:borrow_record_id])
  rescue ActiveRecord::RecordNotFound
    render json: {
      success: false,
      message: "Borrow record not found"
    }, status: :not_found
  end

  def paginate(scope)
    page = (params[:page] || 1).to_i
    per_page = [[(params[:per_page] || 20).to_i, 50].min, 1].max
    scope.page(page).per(per_page)
  end

  def serialize_return_request(return_request)
    {
      id: return_request.id,
      borrow_record_id: return_request.borrow_record_id,
      item: {
        id: return_request.borrow_record.inventory_item.id,
        name: return_request.borrow_record.inventory_item.name,
        category: return_request.borrow_record.inventory_item.category
      },
      status: return_request.status,
      condition_note: return_request.condition_note,
      created_at: return_request.created_at,
      confirmed_at: return_request.confirmed_at
    }
  end

  def serialize_return_request_full(return_request)
    serialize_return_request(return_request).merge(
      confirmed_by: return_request.confirmed_by&.email,
      borrow_date: return_request.borrow_record.borrowed_at,
      due_date: return_request.borrow_record.due_date.to_s
    )
  end

  def serialize_return_request_admin(return_request)
    {
      id: return_request.id,
      member: {
        id: return_request.member.id,
        name: return_request.member.name,
        login_id: return_request.member.login_id,
        email: return_request.member.email
      },
      borrow_record: {
        id: return_request.borrow_record.id,
        borrowed_at: return_request.borrow_record.borrowed_at,
        due_date: return_request.borrow_record.due_date.to_s,
        status: return_request.borrow_record.status,
        quantity: return_request.borrow_record.quantity
      },
      item: {
        id: return_request.borrow_record.inventory_item.id,
        name: return_request.borrow_record.inventory_item.name,
        category: return_request.borrow_record.inventory_item.category
      },
      status: return_request.status,
      condition_note: return_request.condition_note,
      confirmed_by: return_request.confirmed_by&.email,
      created_at: return_request.created_at,
      confirmed_at: return_request.confirmed_at
    }
  end
end
