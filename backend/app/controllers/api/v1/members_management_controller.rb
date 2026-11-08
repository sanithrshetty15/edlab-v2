class Api::V1::MembersManagementController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_admin!

  # GET /api/v1/members-management
  def index
    members = Member.all
    members = filter_members(members)

    paginated = paginate(members.order(created_at: :desc))

    render json: {
      success: true,
      members: paginated.items.map { |m| serialize_member_summary(m) },
      pagination: {
        page: paginated.current_page,
        per_page: paginated.per_page,
        total_count: paginated.total_count,
        total_pages: paginated.total_pages
      }
    }, status: :ok
  end

  # GET /api/v1/members-management/:id
  def show
    member = Member.includes(:user).find(params[:id])

    render json: {
      success: true,
      member: serialize_member_full(member)
    }, status: :ok
  end

  # PATCH /api/v1/members-management/:id/status
  def update_status
    member = Member.find(params[:id])
    new_status = params[:status].to_s.downcase

    unless %w[active inactive deactivated].include?(new_status)
      return render json: {
        success: false,
        message: "Invalid status. Must be one of: active, inactive, deactivated"
      }, status: :unprocessable_entity
    end

    if member.status == new_status
      return render json: {
        success: false,
        message: "Member already has this status"
      }, status: :unprocessable_entity
    end

    previous_status = member.status
    member.update!(status: new_status)

    AuditLogService.log_member_status_change(
      member,
      changed_by: current_user,
      new_status: new_status,
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Member status updated to #{new_status}",
      member: {
        id: member.id,
        login_id: member.login_id,
        name: member.name,
        email: member.email,
        status: member.status,
        previous_status: previous_status
      }
    }, status: :ok
  end

  # PATCH /api/v1/members-management/:id/_fields
  def update_official_fields
    member = Member.find(params[:id])
    changes = {}

    if params[:name].present? && member.name != params[:name]
      changes[:name] = { from: member.name, to: params[:name] }
      member.update!(name: params[:name])
    end

    if params[:usn].present?
      usn = params[:usn].to_s.strip.upcase
      if member.usn != usn
        validate_usn_uniqueness(usn, member)
        changes[:usn] = { from: member.usn, to: usn }
        member.update!(usn: usn)
      end
    end

    if params[:roll_number].present?
      roll = params[:roll_number].to_s.strip
      if member.roll_number != roll
        validate_roll_number_uniqueness(roll, member)
        changes[:roll_number] = { from: member.roll_number, to: roll }
        member.update!(roll_number: roll)
      end
    end

    if params[:academic_year].present? && member.academic_year != params[:academic_year].to_i
      ay = params[:academic_year].to_i
      unless ay.between?(1, 8)
        return render json: {
          success: false,
          message: "Academic year must be between 1 and 8"
        }, status: :unprocessable_entity
      end
      changes[:academic_year] = { from: member.academic_year, to: ay }
      member.update!(academic_year: ay)
    end

    if params[:branch].present? && member.branch != params[:branch]
      changes[:branch] = { from: member.branch, to: params[:branch] }
      member.update!(branch: params[:branch])
    end

    if params[:section].present? && member.section != params[:section]
      changes[:section] = { from: member.section, to: params[:section] }
      member.update!(section: params[:section])
    end

    if params[:phone].present? && member.phone != params[:phone]
      phone = params[:phone].to_s.strip
      unless phone.match?(/\A\+?[1-9]\d{7,14}\z/)
        return render json: {
          success: false,
          message: "Invalid phone number format"
        }, status: :unprocessable_entity
      end
      
      if User.find_by(phone: phone, id: member.user.id).nil?
        existing_user = User.find_by(phone: phone)
        if existing_user && existing_user.member&.active? && existing_user.id != member.user.id
          return render json: {
            success: false,
            message: "This phone number is already in use"
          }, status: :unprocessable_entity
        end
      end
      
      changes[:phone] = { from: member.phone, to: phone }
      member.update!(phone: phone)
      member.user.update!(phone: phone)
    end

    if changes.any?
      AuditLogService.log!(
        user: current_user,
        action: "official_fields_updated",
        resource_type: "Member",
        resource_id: member.id,
        details: changes,
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Official fields updated successfully",
        member: serialize_member(member)
      }, status: :ok
    else
      render json: {
        success: false,
        message: "No changes provided"
      }, status: :unprocessable_entity
    end
  end

  # PATCH /api/v1/members-management/:id/force-password-reset
  def force_password_reset
    member = Member.find(params[:id])

    member.update!(must_change_password: true)

    AuditLogService.log!(
      user: current_user,
      action: "password_reset_forced",
      resource_type: "Member",
      resource_id: member.id,
      details: {
        member_login_id: member.login_id,
        member_name: member.name
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Member will be required to change password on next login"
    }, status: :ok
  end

  # DELETE /api/v1/members-management/:id
  def destroy
    member = Member.find(params[:id])

    member.update!(
      status: "deactivated",
      deleted_at: Time.current
    )

    AuditLogService.log!(
      user: current_user,
      action: "member_deleted",
      resource_type: "Member",
      resource_id: member.id,
      details: {
        login_id: member.login_id,
        name: member.name,
        email: member.email
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Member account deactivated successfully"
    }, status: :ok
  end

  private

  def require_admin!
    unless current_user.admin?
      return render json: {
        success: false,
        message: "Administrator access required"
      }, status: :forbidden
    end
  end

  def filter_members(scope)
    scope = scope.where(status: params[:status]) if params[:status].present?
    scope = scope.where(academic_year: params[:academic_year].to_i) if params[:academic_year].present?
    scope = scope.where(branch: params[:branch]) if params[:branch].present?
    scope = scope.where("name ILIKE ?", "%#{params[:search]}%") if params[:search].present?
    scope = scope.where("login_id ILIKE ?", "%#{params[:search]}%") if params[:search].present? && params[:search_field] == "login_id"
    scope
  end

  def paginate(scope)
    page = (params[:page] || 1).to_i
    per_page = [[(params[:per_page] || 20).to_i, 100].min, 1].max
    scope.page(page).per(per_page)
  end

  def validate_usn_uniqueness(usn, current_member)
    existing = Member.active.where.not(id: current_member.id).find_by(usn: usn)
    raise "This USN is already in use by another member" if existing
  end

  def validate_roll_number_uniqueness(roll, current_member)
    return if roll.blank?
    existing = Member.active.where.not(id: current_member.id).find_by(roll_number: roll)
    raise "This roll number is already in use by another member" if existing
  end

  def serialize_member_summary(member)
    {
      id: member.id,
      name: member.name,
      login_id: member.login_id,
      usn: member.usn,
      roll_number: member.roll_number,
      academic_year: member.academic_year,
      branch: member.branch,
      section: member.section,
      email: member.email,
      phone: member.phone,
      phone_verified: member.phone_verified,
      status: member.status,
      must_change_password: member.must_change_password,
      created_at: member.created_at,
      updated_at: member.updated_at
    }
  end

  def serialize_member(member)
    serialize_member_summary(member).merge(
      github_url: member.github_url,
      linkedin_url: member.linkedin_url,
      portfolio_url: member.portfolio_url,
      bio: member.bio,
      interests: member.interests,
      login_id_changed: member.login_id_changed,
      login_id_changed_at: member.login_id_changed_at
    )
  end

  def serialize_member_full(member)
    serialize_member(member).merge(
      user: {
        id: member.user.id,
        email: member.user.email,
        active: member.user.active?,
        role: member.user.role
      }
    )
  end
end
