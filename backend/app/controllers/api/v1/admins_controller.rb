class Api::V1::AdminsController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_primary_admin!

  # GET /api/v1/admins
  def index
    admins = Admin.includes(:user).order(created_at: :desc)

    render json: {
      success: true,
      admins: admins.map { |admin| serialize_admin(admin) }
    }, status: :ok
  end

  # GET /api/v1/admins/:id
  def show
    admin = Admin.includes(:user).find(params[:id])

    render json: {
      success: true,
      admin: serialize_admin(admin)
    }, status: :ok
  end

  # POST /api/v1/admins
  def create
    result = AdminCreationService.create(
      params: admin_params,
      created_by: current_user,
      ip_address: request.remote_ip
    )

    if result[:success]
      AuditLogService.log_admin_created(
        result[:admin],
        created_by: current_user,
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Admin account created successfully",
        admin: serialize_admin(result[:admin])
      }, status: :created
    else
      render json: {
        success: false,
        message: result[:error]
      }, status: result[:status] || :unprocessable_entity
    end
  end

  # PATCH /api/v1/admins/:id
  def update
    admin = Admin.find(params[:id])
    changes = {}

    if params[:name].present? && admin.name != params[:name]
      changes[:name] = { from: admin.name, to: params[:name] }
      admin.update!(name: params[:name])
    end

    if params[:is_primary].present? && admin.is_primary != params[:is_primary]
      if admin.is_primary && !params[:is_primary]
        validate_primary_admin_transfer(admin)
      elsif !admin.is_primary && params[:is_primary]
        validate_primary_admin_limit
      end
      changes[:is_primary] = { from: admin.is_primary, to: params[:is_primary] }
      admin.update!(is_primary: params[:is_primary])
    end

    if changes.any?
      AuditLogService.log_admin_updated(
        admin,
        updated_by: current_user,
        changes: changes,
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Admin updated successfully",
        admin: serialize_admin(admin)
      }, status: :ok
    else
      render json: {
        success: false,
        message: "No changes provided"
      }, status: :unprocessable_entity
    end
  end

  # DELETE /api/v1/admins/:id
  def destroy
    admin = Admin.find(params[:id])

    if admin.id == current_user.admin.id
      return render json: {
        success: false,
        message: "You cannot remove your own admin account"
      }, status: :forbidden
    end

    admin_id = admin.id
    admin_name = admin.name
    admin_email = admin.user.email

    ActiveRecord::Base.transaction do
      admin.user.update!(active: false)
      admin.destroy!
    end

    AuditLogService.log_admin_removed(
      admin,
      removed_by: current_user,
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Admin account removed successfully"
    }, status: :ok
  end

  # POST /api/v1/admins/invite
  def invite
    result = AdminInvitationService.invite(
      params: invite_params,
      invited_by: current_user,
      ip_address: request.remote_ip
    )

    if result[:success]
      AuditLogService.log_membership_invite(
        invited_by: current_user,
        member_data: result[:member_data],
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Invitation sent successfully",
        invitation: {
          email: result[:member_data][:email],
          temp_login_id: result[:temp_login_id],
          expires_at: result[:expires_at]
        }
      }, status: :created
    else
      render json: {
        success: false,
        message: result[:error]
      }, status: result[:status] || :unprocessable_entity
    end
  end

  private

  def require_primary_admin!
    unless current_user.primary_admin?
      return render json: {
        success: false,
        message: "Only primary admins can perform this action"
      }, status: :forbidden
    end
  end

  def admin_params
    params.permit(:name, :email, :is_primary, :phone)
  end

  def invite_params
    params.permit(:name, :email, :phone, :academic_year, :branch, :section, :usn, :roll_number, :password)
  end

  def serialize_admin(admin)
    {
      id: admin.id,
      name: admin.name,
      email: admin.user.email,
      phone: admin.user.phone,
      is_primary: admin.is_primary,
      created_at: admin.created_at,
      user: {
        id: admin.user.id,
        active: admin.user.active?
      }
    }
  end

  def validate_primary_admin_limit
    existing_primary = Admin.primary.count
    if existing_primary >= 1 && Admin.primary.first.id != current_user.admin.id
      raise "Only one primary admin is allowed"
    end
  end

  def validate_primary_admin_transfer(admin)
    raise "Cannot remove the only primary admin" if Admin.primary.count == 1 && admin.is_primary
  end
end
