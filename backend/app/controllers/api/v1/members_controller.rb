class Api::V1::MembersController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!, except: [:update_profile, :accept_lab_rules]

  # POST /api/v1/members/register
  def register
    result = MemberRegistrationService.register(
      params: registration_params,
      ip_address: request.remote_ip,
      user_agent: request.user_agent
    )

    if result[:success]
      AuditLogService.log!(
        user: nil,
        action: "member_registered",
        resource_type: "Member",
        resource_id: result[:member].id,
        details: { login_id: result[:member].login_id, academic_year: result[:member].academic_year },
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Registration successful. Please check your email for verification.",
        member: serialize_member(result[:member]),
        user: {
          id: result[:user].id,
          email: result[:user].email,
          role: result[:user].role
        }
      }, status: :created
    else
      render json: {
        success: false,
        message: result[:error]
      }, status: result[:status] || :unprocessable_entity
    end
  end

  # GET /api/v1/members/me
  def show_current
    member = current_user.member
    raise ActiveRecord::RecordNotFound unless member

    render json: {
      success: true,
      member: serialize_member_full(member)
    }, status: :ok
  end

  # PATCH /api/v1/members/me/profile
  def update_profile
    member = current_user.member
    raise ActiveRecord::RecordNotFound unless member

    permitted = profile_params

    if member.update!(permitted)
      AuditLogService.log!(
        user: current_user,
        action: "profile_updated",
        resource_type: "Member",
        resource_id: member.id,
        details: permitted.transform_keys(&:to_s),
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Profile updated successfully",
        member: serialize_member(member)
      }, status: :ok
    else
      render json: {
        success: false,
        message: "Profile update failed",
        errors: member.errors.full_messages
      }, status: :unprocessable_entity
    end
  end

  # PATCH /api/v1/members/me/accept-rules
  def accept_lab_rules
    member = current_user.member
    raise ActiveRecord::RecordNotFound unless member

    version = LabRulesVersion.current
    raise "No current lab rules version found" unless version

    if member.lab_rules_acceptances.exists?(lab_rules_version_id: version.id)
      return render json: {
        success: false,
        message: "Lab rules already accepted"
      }, status: :unprocessable_entity
    end

    member.lab_rules_acceptances.create!(
      lab_rules_version_id: version.id,
      accepted_at: Time.current
    )

    AuditLogService.log!(
      user: current_user,
      action: "lab_rules_accepted",
      resource_type: "Member",
      resource_id: member.id,
      details: { version_number: version.version_number },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Lab rules accepted successfully"
    }, status: :ok
  end

  # PATCH /api/v1/members/me/phone-verification
  def verify_phone
    member = current_user.member
    raise ActiveRecord::RecordNotFound unless member

    otp_code = params[:otp_code].to_s.strip
    delivery_method = params[:delivery_method].to_s

    unless otp_code.present? && delivery_method == "sms"
      return render json: {
        success: false,
        message: "OTP code and SMS delivery method are required"
      }, status: :unprocessable_entity
    end

    unless member.phone.present?
      return render json: {
        success: false,
        message: "No phone number on file to verify"
      }, status: :unprocessable_entity
    end

    begin
      approved = TwilioService.check_verification(
        phone_number: member.phone,
        code: otp_code
      )
    rescue ArgumentError => e
      return render json: {
        success: false,
        message: "Invalid or expired verification code"
      }, status: :unauthorized
    rescue StandardError => e
      Rails.logger.error("Phone verification check failed: #{e.class} - #{e.message}")
      return render json: {
        success: false,
        message: "Unable to verify code"
      }, status: :unprocessable_entity
    end

    unless approved
      return render json: {
        success: false,
        message: "Invalid or expired verification code"
      }, status: :unauthorized
    end

    member.user.otp_codes
      .where(purpose: "phone_verification", delivery_method: "sms", used: false)
      .update_all(used: true)

    member.update!(phone_verified: true)

    AuditLogService.log!(
      user: current_user,
      action: "phone_verified",
      resource_type: "Member",
      resource_id: member.id,
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Phone number verified successfully"
    }, status: :ok
  end

  # POST /api/v1/members/phone-verification/request
  def request_phone_verification
    member = current_user.member
    raise ActiveRecord::RecordNotFound unless member

    unless member.phone.present?
      return render json: {
        success: false,
        message: "No phone number on file to verify"
      }, status: :unprocessable_entity
    end

    begin
      result = OtpDeliveryService.send_sms(
        user: current_user,
        purpose: "phone_verification"
      )

      if result[:success]
        render json: {
          success: true,
          message: "Verification code sent to your phone"
        }, status: :ok
      else
        render json: {
          success: false,
          message: result[:message]
        }, status: :unprocessable_entity
      end
    rescue => e
      Rails.logger.error("Phone verification request failed: #{e.message}")
      render json: {
        success: false,
        message: "Unable to send verification code"
      }, status: :unprocessable_entity
    end
  end

  private

  def registration_params
    params.permit(
      :name,
      :usn,
      :roll_number,
      :academic_year,
      :branch,
      :section,
      :email,
      :phone,
      :password,
      :password_confirmation,
      :login_id,
      :accept_lab_rules
    )
  end

  def profile_params
    params.permit(
      :github_url,
      :linkedin_url,
      :portfolio_url,
      :bio,
      interests: []
    )
  end

  def serialize_member(member)
    {
      id: member.id,
      name: member.name,
      usn: member.usn,
      roll_number: member.roll_number,
      login_id: member.login_id,
      academic_year: member.academic_year,
      branch: member.branch,
      section: member.section,
      email: member.email,
      phone: member.phone_phone_verified? ? member.phone : nil,
      phone_verified: member.phone_verified,
      github_url: member.github_url,
      linkedin_url: member.linkedin_url,
      portfolio_url: member.portfolio_url,
      bio: member.bio,
      interests: member.interests,
      status: member.status,
      must_change_password: member.must_change_password,
      created_at: member.created_at
    }
  end

  def serialize_member_full(member)
    serialize_member(member).merge(
      login_id_changed: member.login_id_changed,
      login_id_changed_at: member.login_id_changed_at
    )
  end
end
