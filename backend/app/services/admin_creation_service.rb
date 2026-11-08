class AdminCreationService
  MAX_ADMIN_COUNT = 3
  MAX_PRIMARY_COUNT = 1

  class AdminCreationError < StandardError; end

  def self.create(params:, created_by:, ip_address: nil)
    ActiveRecord::Base.transaction do
      email = params[:email].to_s.strip.downcase
      name = params[:name].to_s.strip
      phone = params[:phone].to_s.strip
      password = params[:password].to_s
      is_primary = params[:is_primary].to_s == "true" || params[:is_primary] == true

      validate_admin_creation(
        email: email,
        name: name,
        phone: phone,
        password: password,
        is_primary: is_primary,
        current_primary_count: Admin.primary.count
      )

      user = User.create!(
        email: email,
        phone: phone.presence,
        password: password,
        password_confirmation: password,
        role: "admin",
        active: true
      )

      admin = Admin.create!(
        user: user,
        name: name,
        is_primary: is_primary
      )

      AuditLogService.log!(
        user: created_by,
        action: "admin_created",
        resource_type: "Admin",
        resource_id: admin.id,
        details: {
          admin_name: name,
          admin_email: email,
          is_primary: is_primary
        },
        ip_address: ip_address
      )

      {
        success: true,
        admin: admin,
        user: user
      }
    end
  rescue AdminCreationError => e
    {
      success: false,
      error: e.message,
      status: :unprocessable_entity
    }
  rescue ActiveRecord::RecordInvalid => e
    {
      success: false,
      error: e.message,
      status: :unprocessable_entity
    }
  rescue => e
    Rails.logger.error("Admin creation failed: #{e.class} - #{e.message}")
    {
      success: false,
      error: "Failed to create admin account",
      status: :internal_server_error
    }
  end

  def self.validate_admin_creation(email:, name:, phone:, password:, is_primary:, current_primary_count:)
    raise AdminCreationError, "Admin name is required" if name.blank?
    raise AdminCreationError, "Admin email is required" if email.blank?
    raise AdminCreationError, "Password is required" if password.blank?
    raise AdminCreationError, "Password must be at least 8 characters" if password.length < 8

    unless email.match?(/\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i)
      raise AdminCreationError, "Please provide a valid email address"
    end

    if phone.present? && !phone.match?(/\A\+?[1-9]\d{7,14}\z/)
      raise AdminCreationError, "Please provide a valid phone number"
    end

    if is_primary && current_primary_count >= MAX_PRIMARY_COUNT
      raise AdminCreationError, "Cannot create another primary admin. Maximum of one primary admin allowed."
    end

    existing_admin = Admin.find_by(user_id: User.find_by(email: email)&.id)
    if existing_admin
      raise AdminCreationError, "An admin account with this email already exists"
    end

    existing_user = User.find_by(email: email)
    if existing_user
      raise AdminCreationError, "An account with this email already exists"
    end

    total_admins = Admin.count
    if total_admins >= MAX_ADMIN_COUNT
      raise AdminCreationError, "Maximum of #{MAX_ADMIN_COUNT} administrators allowed"
    end
  end
end
