class Api::V1::InventoryController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :require_admin_or_member!

  # GET /api/v1/inventory
  def index
    scope = InventoryItem
      .where(is_active: true, deleted_at: nil)
      .where("available_quantity > 0")
      .order(name: :asc)

    if params[:category].present?
      scope = scope.where(category: params[:category])
    end

    if params[:search].present?
      scope = scope.where("name ILIKE ? OR description ILIKE ?", "%#{params[:search]}%", "%#{params[:search]}%")
    end

    if params[:in_stock].present? && params[:in_stock] == "false"
      scope = InventoryItem
        .where(is_active: true, deleted_at: nil)
        .order(name: :asc)
    end

    items = paginate(scope)

    render json: {
      success: true,
      items: items.items.map { |i| serialize_inventory_item(i) },
      pagination: {
        page: items.current_page,
        per_page: items.per_page,
        total_count: items.total_count
      }
    }, status: :ok
  end

  # GET /api/v1/inventory/:id
  def show
    item = InventoryItem.find_by(id: params[:id], deleted_at: nil)

    unless item
      return render json: {
        success: false,
        message: "Inventory item not found"
      }, status: :not_found
    end

    borrow_history = item.borrow_records
      .includes(:member)
      .order(borrowed_at: :desc)
      .limit(20)

    render json: {
      success: true,
      item: serialize_inventory_item_full(item),
      borrow_history: borrow_history.map { |b| serialize_borrow_record(b) }
    }, status: :ok
  end

  # POST /api/v1/inventory
  def create
    raise "Administrator access required" unless current_user.admin?

    name = params[:name].to_s.strip
    category = params[:category].to_s.strip
    description = params[:description].to_s.strip
    total_quantity = params[:total_quantity].to_i
    storage_location = params[:storage_location].to_s.strip
    condition = params[:condition].to_s.strip

    validate_inventory_item_inputs(
      name: name,
      category: category,
      total_quantity: total_quantity
    )

    ActiveRecord::Base.transaction do
      item = InventoryItem.create!(
        name: name,
        category: category,
        description: description.presence,
        total_quantity: total_quantity,
        available_quantity: total_quantity,
        storage_location: storage_location.presence,
        condition: condition.presence
      )

      AuditLogService.log!(
        user: current_user,
        action: "inventory_item_created",
        resource_type: "InventoryItem",
        resource_id: item.id,
        details: {
          name: name,
          category: category,
          total_quantity: total_quantity
        },
        ip_address: request.remote_ip
      )
    end

    render json: {
      success: true,
      message: "Inventory item created",
      item: serialize_inventory_item_full(item)
    }, status: :created
  end

  # PATCH /api/v1/inventory/:id
  def update
    raise "Administrator access required" unless current_user.admin?

    item = InventoryItem.find_by(id: params[:id], deleted_at: nil)

    unless item
      return render json: {
        success: false,
        message: "Inventory item not found"
      }, status: :not_found
    end

    changes = {}

    if params[:name].present? && item.name != params[:name].to_s.strip
      name = params[:name].to_s.strip
      unless name.present?
        return render json: {
          success: false,
          message: "Name cannot be empty"
        }, status: :unprocessable_entity
      end
      changes[:name] = { from: item.name, to: name }
      item.update!(name: name)
    end

    if params[:category].present? && item.category != params[:category].to_s.strip
      category = params[:category].to_s.strip
      unless category.present?
        return render json: {
          success: false,
          message: "Category cannot be empty"
        }, status: :unprocessable_entity
      end
      changes[:category] = { from: item.category, to: category }
      item.update!(category: category)
    end

    if params[:description].present? && item.description != params[:description].to_s.strip
      changes[:description] = { from: item.description.truncate(100), to: params[:description].truncate(100) }
      item.update!(description: params[:description].to_s.strip.presence)
    end

    if params[:total_quantity].present? && item.total_quantity != params[:total_quantity].to_i
      total_qty = params[:total_quantity].to_i
      unless total_qty >= 0
        return render json: {
          success: false,
          message: "Total quantity must be non-negative"
        }, status: :unprocessable_entity
      end

      current_available = item.available_quantity
      new_available = [current_available, total_qty].min
      changes[:total_quantity] = { from: item.total_quantity, to: total_qty }
      item.update!(total_quantity: total_qty, available_quantity: new_available)
    end

    if params[:storage_location].present? && item.storage_location != params[:storage_location].to_s.strip
      changes[:storage_location] = { from: item.storage_location, to: params[:storage_location] }
      item.update!(storage_location: params[:storage_location].to_s.strip.presence)
    end

    if params[:condition].present? && item.condition != params[:condition].to_s.strip
      changes[:condition] = { from: item.condition, to: params[:condition] }
      item.update!(condition: params[:condition].to_s.strip.presence)
    end

    if changes.any?
      AuditLogService.log!(
        user: current_user,
        action: "inventory_item_updated",
        resource_type: "InventoryItem",
        resource_id: item.id,
        details: changes,
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Inventory item updated",
        item: serialize_inventory_item_full(item)
      }, status: :ok
    else
      render json: {
        success: false,
        message: "No changes provided"
      }, status: :unprocessable_entity
    end
  end

  # DELETE /api/v1/inventory/:id
  def destroy
    raise "Administrator access required" unless current_user.admin?

    item = InventoryItem.find_by(id: params[:id], deleted_at: nil)

    unless item
      return render json: {
        success: false,
        message: "Inventory item not found"
      }, status: :not_found
    end

    item.update!(is_active: false, deleted_at: Time.current)

    AuditLogService.log!(
      user: current_user,
      action: "inventory_item_deleted",
      resource_type: "InventoryItem",
      resource_id: item.id,
      details: {
        name: item.name,
        category: item.category,
        total_quantity: item.total_quantity
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Inventory item removed from active inventory"
    }, status: :ok
  end

  # POST /api/v1/inventory/import
  def import
    raise "Administrator access required" unless current_user.admin?

    file = params[:file]
    unless file.present?
      return render json: {
        success: false,
        message: "File is required"
      }, status: :unprocessable_entity
    end

    unless file.content_type.in?(%w[application/vnd.ms-excel application/vnd.openxmlformats-officedocument.spreadsheetml.sheet text/csv])
      return render json: {
        success: false,
        message: "Only Excel (.xlsx, .xls) or CSV files are supported"
      }, status: :unprocessable_entity
    end

    result = ImportInventoryService.import(
      file: file,
      imported_by: current_user
    )

    if result[:success]
      AuditLogService.log_inventory_import(
        result[:import_record],
        imported_by: current_user
      )

      render json: {
        success: true,
        message: "Inventory imported successfully",
        import: {
          id: result[:import_record].id,
          filename: result[:import_record].filename,
          total_rows: result[:import_record].total_rows,
          imported_rows: result[:import_record].imported_rows,
          error_rows: result[:import_record].error_rows,
          status: result[:import_record].status
        }
      }, status: :created
    else
      render json: {
        success: false,
        message: result[:error]
      }, status: result[:status] || :unprocessable_entity
    end
  end

  # GET /api/v1/inventory/stats
  def stats
    raise "Administrator access required" unless current_user.admin?

    total_items = InventoryItem.where(is_active: true, deleted_at: nil).count
    total_stock = InventoryItem.where(is_active: true, deleted_at: nil).sum(:total_quantity)
    available_stock = InventoryItem.where(is_active: true, deleted_at: nil).sum(:available_quantity)
    borrowed_stock = total_stock - available_stock

    category_breakdown = InventoryItem
      .where(is_active: true, deleted_at: nil)
      .group(:category)
      .select("category, COUNT(*) AS items, SUM(total_quantity) AS total, SUM(available_quantity) AS available")
      .order("total DESC")

    render json: {
      success: true,
      stats: {
        total_items: total_items,
        total_stock: total_stock,
        available_stock: available_stock,
        borrowed_stock: borrowed_stock,
        categories: category_breakdown.map { |c|
          {
            category: c.category,
            items: c.items,
            total_quantity: c.total,
            available_quantity: c.available
          }
        }
      }
    }, status: :ok
  end

  private

  def require_admin_or_member!
    unless current_user.admin? || (current_user.member? && current_user.member.status == "active")
      return render json: {
        success: false,
        message: "Access denied"
      }, status: :forbidden
    end
  end

  def validate_inventory_item_inputs(name:, category:, total_quantity:)
    raise "Name is required" if name.blank?
    raise "Category is required" if category.blank?
    raise "Total quantity is required" if total_quantity.blank?
    raise "Total quantity must be non-negative" unless total_quantity >= 0

    existing = InventoryItem.where(is_active: true, deleted_at: nil)
      .find_by(name: name, category: category)
    if existing
      raise "An item with this name and category already exists"
    end
  end


  def serialize_inventory_item(item)
    {
      id: item.id,
      name: item.name,
      category: item.category,
      total_quantity: item.total_quantity,
      available_quantity: item.available_quantity,
      storage_location: item.storage_location,
      condition: item.condition,
      is_available: item.available_quantity > 0,
      created_at: item.created_at
    }
  end

  def serialize_inventory_item_full(item)
    serialize_inventory_item(item).merge(
      description: item.description,
      updated_at: item.updated_at
    )
  end

  def serialize_borrow_record(borrow)
    {
      id: borrow.id,
      member: {
        id: borrow.member.id,
        name: borrow.member.name,
        login_id: borrow.member.login_id
      },
      borrowed_at: borrow.borrowed_at,
      due_date: borrow.due_date.to_s,
      status: borrow.status,
      returned_at: borrow.returned_at,
      approved_by: borrow.approved_by&.email
    }
  end
end
