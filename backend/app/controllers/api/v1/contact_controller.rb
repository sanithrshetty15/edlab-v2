class Api::V1::ContactController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :require_admin!, only: [:index, :show, :update_status, :destroy]
  before_action :require_primary_admin!, only: [:update_contact_email, :verify_contact_email_change]

  # POST /api/v1/contact/messages
  def create
    member = current_user.member
    raise "Active member account required" unless member && member.status == "active"

    subject = params[:subject].to_s.strip
    category = params[:category].to_s.strip
    message = params[:message].to_s.strip

    unless subject.present?
      return render json: {
        success: false,
        message: "Subject is required"
      }, status: :unprocessable_entity
    end

    unless category.present?
      return render json: {
        success: false,
        message: "Category is required"
      }, status: :unprocessable_entity
    end

    unless message.present?
      return render json: {
        success: false,
        message: "Message is required"
      }, status: :unprocessable_entity
    end

    unless %w[general inquiry feedback complaint suggestion other].include?(category.downcase)
      return render json: {
        success: false,
        message: "Invalid category. Must be one of: general, inquiry, feedback, complaint, suggestion, other"
      }, status: :unprocessable_entity
    end

    if message.length > 5000
      return render json: {
        success: false,
        message: "Message is too long (max 5000 characters)"
      }, status: :unprocessable_entity
    end

    contact_message = ContactMessage.create!(
      member_id: member.id,
      subject: subject,
      category: category.downcase,
      message: message,
      status: "new"
    )

    AuditLogService.log!(
      user: current_user,
      action: "contact_message_sent",
      resource_type: "ContactMessage",
      resource_id: contact_message.id,
      details: {
        subject: subject,
        category: category.downcase
      },
      ip_address: request.remote_ip
    )

    NotificationService.notify_contact_message_received(contact_message)

    render json: {
      success: true,
      message: "Your message has been sent successfully",
      contact_message: {
        id: contact_message.id,
        subject: contact_message.subject,
        category: contact_message.category,
        status: contact_message.status,
        created_at: contact_message.created_at
      }
    }, status: :created
  end

  # GET /api/v1/contact/messages
  def index
    scope = ContactMessage
      .includes(:member)
      .order(created_at: :desc)

    if params[:status].present?
      scope = scope.where(status: params[:status])
    end

    if params[:category].present?
      scope = scope.where(category: params[:category])
    end

    if params[:search].present?
      scope = scope.where("subject ILIKE ? OR message ILIKE ?", "%#{params[:search]}%", "%#{params[:search]}%")
    end

    messages = paginate(scope)

    render json: {
      success: true,
      messages: messages.items.map { |m| serialize_contact_message(m) },
      pagination: {
        page: messages.current_page,
        per_page: messages.per_page,
        total_count: messages.total_count,
        new_count: ContactMessage.where(status: "new").count
      }
    }, status: :ok
  end

  # GET /api/v1/contact/messages/:id
  def show
    message = ContactMessage.find(params[:id])

    render json: {
      success: true,
      message_data: serialize_contact_message_full(message)
    }, status: :ok
  end

  # PATCH /api/v1/contact/messages/:id/status
  def update_status
    message = ContactMessage.find(params[:id])

    new_status = params[:status].to_s.downcase

    unless %w[new in_progress resolved closed].include?(new_status)
      return render json: {
        success: false,
        message: "Invalid status. Must be one of: new, in_progress, resolved, closed"
      }, status: :unprocessable_entity
    end

    if message.status == new_status
      return render json: {
        success: false,
        message: "Message already has this status"
      }, status: :unprocessable_entity
    end

    message.update!(status: new_status)

    AuditLogService.log!(
      user: current_user,
      action: "contact_message_status_updated",
      resource_type: "ContactMessage",
      resource_id: message.id,
      details: {
        subject: message.subject,
        previous_status: message.status,
        new_status: new_status
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Status updated to #{new_status}",
      contact_message: serialize_contact_message(message)
    }, status: :ok
  end

  # DELETE /api/v1/contact/messages/:id
  def destroy
    message = ContactMessage.find(params[:id])

    message.update!(status: "closed")

    render json: {
      success: true,
      message: "Contact message closed"
    }, status: :ok
  end

  # POST /api/v1/contact/update-email/request
  def request_contact_email_change
    current_email = ENV.fetch("GMAIL_USERNAME", "")
    new_email = params[:new_email].to_s.strip.downcase

    unless new_email.present?
      return render json: {
        success: false,
        message: "New email address is required"
      }, status: :unprocessable_entity
    end

    unless new_email.match?(/\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i)
      return render json: {
        success: false,
        message: "Please provide a valid email address"
      }, status: :unprocessable_entity
    end

    if new_email == current_email
      return render json: {
        success: false,
        message: "New email must be different from current email"
      }, status: :unprocessable_entity
    end

    thirty_days_ago = 30.days.ago

    recent_change = PlatformSetting
      .where(key: "contact_email_change_requested_at")
      .where("value > ?", thirty_days_ago.to_s)
      .first

    if recent_change
      return render json: {
        success: false,
        message: "Contact email can only be changed once every 30 days"
      }, status: :unprocessable_entity
    end

    PlatformSetting.create!(
      key: "contact_email_change_pending",
      value: new_email
    )

    PlatformSetting.where(key: "contact_email_change_requested_at").first_or_create!
      .update!(value: Time.current.to_s)

    OtpDeliveryService.send_email(
      user: current_user,
      purpose: "contact_email_change"
    )

    render json: {
      success: true,
      message: "Verification code sent to your email"
    }, status: :ok
  end

  # POST /api/v1/contact/update-email/verify
  def verify_contact_email_change
    new_email = params[:new_email].to_s.strip.downcase
    otp_code = params[:otp_code].to_s.strip

    unless new_email.present? && otp_code.present?
      return render json: {
        success: false,
        message: "New email and verification code are required"
      }, status: :unprocessable_entity
    end

    pending_setting = PlatformSetting.find_by(key: "contact_email_change_pending")
    unless pending_setting&.value == new_email
      return render json: {
        success: false,
        message: "No pending email change found for this address"
      }, status: :unprocessable_entity
    end

    user_otp = current_user.otp_codes
      .where(purpose: "contact_email_change", delivery_method: "email")
      .unused
      .unexpired
      .order(created_at: :desc)
      .first

    unless user_otp&.verify_code(otp_code)
      return render json: {
        success: false,
        message: "Invalid or expired verification code"
      }, status: :unauthorized
    end

    previous_email = ENV.fetch("GMAIL_USERNAME", "")

    ENV["GMAIL_USERNAME"] = new_email

    pending_setting.destroy!

    AuditLogService.log_contact_email_change(
      previous_email: previous_email,
      new_email: new_email,
      changed_by: current_user,
      ip_address: request.remote_ip
    )

    NotificationService.notify!(
      user: current_user,
      title: "Contact email updated",
      body: "Your contact email has been changed from #{previous_email} to #{new_email}",
      type: "administrative_update"
    )

    render json: {
      success: true,
      message: "Contact email changed successfully to #{new_email}",
      new_email: new_email
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

  def require_primary_admin!
    unless current_user.primary_admin?
      return render json: {
        success: false,
        message: "Primary admin access required"
      }, status: :forbidden
    end
  end


  def serialize_contact_message(message)
    {
      id: message.id,
      member: {
        id: message.member.id,
        name: message.member.name,
        login_id: message.member.login_id
      },
      subject: message.subject,
      category: message.category,
      status: message.status,
      created_at: message.created_at,
      updated_at: message.updated_at
    }
  end

  def serialize_contact_message_full(message)
    serialize_contact_message(message).merge(
      message: message.message,
      email_delivery_status: message.email_delivery_status
    )
  end
end
