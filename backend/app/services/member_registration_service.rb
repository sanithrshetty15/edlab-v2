class MemberRegistrationService
  class RegistrationError < StandardError; end

  def self.register(params:, ip_address: nil, user_agent: nil)
    ActiveRecord::Base.transaction do
      login_id = params[:login_id].to_s.strip
      email = params[:email].to_s.strip.downcase
      phone = params[:phone].to_s.strip
      name = params[:name].to_s.strip
      usn = params[:usn].to_s.strip.upcase
      roll_number = params[:roll_number].to_s.strip
      academic_year = params[:academic_year].to_i
      branch = params[:branch].to_s.strip
      section = params[:section].to_s.strip
      password = params[:password].to_s
      password_confirmation = params[:password_confirmation].to_s

      validate_registration_inputs(
        login_id: login_id,
        email: email,
        phone: phone,
        name: name,
        usn: usn,
        roll_number: roll_number,
        academic_year: academic_year,
        branch: branch,
        password: password
      )

      user = User.create!(
        email: email,
        phone: phone,
        password: password,
        password_confirmation: password_confirmation,
        role: "member",
        active: true
      )

      member = Member.create!(
        user: user,
        name: name,
        login_id: login_id,
        usn: usn.presence,
        roll_number: roll_number.presence,
        academic_year: academic_year,
        branch: branch,
        section: section.presence,
        email: email,
        phone: phone,
        must_change_password: true
      )

      if params[:accept_lab_rules]
        version = LabRulesVersion.current
        if version
          member.lab_rules_acceptances.create!(
            lab_rules_version_id: version.id,
            accepted_at: Time.current
          )
        end
      end

      {
        success: true,
        member: member,
        user: user
      }
    end
  rescue ActiveRecord::RecordInvalid => e
    {
      success: false,
      error: e.message,
      status: :unprocessable_entity
    }
  rescue RegistrationError => e
    {
      success: false,
      error: e.message,
      status: :unprocessable_entity
    }
  rescue => e
    Rails.logger.error("Registration failed: #{e.class} - #{e.message}")
    {
      success: false,
      error: "Registration failed. Please try again.",
      status: :internal_server_error
    }
  end

  def self.validate_registration_inputs(login_id:, email:, phone:, name:, usn:, roll_number:, academic_year:, branch:, password:)
    raise RegistrationError, "Login ID is required" if login_id.blank?
    raise RegistrationError, "Name is required" if name.blank?
    raise RegistrationError, "Email is required" if email.blank?
    raise RegistrationError, "Phone number is required" if phone.blank?
    raise RegistrationError, "Academic year is required" if academic_year.blank? || academic_year < 1
    raise RegistrationError, "Branch is required" if branch.blank?
    raise RegistrationError, "Password is required" if password.blank?
    raise RegistrationError, "Password must be at least 8 characters" if password.length < 8

    unless login_id.match?(/\A[\w.-]+\z/)
      raise RegistrationError, "Login ID can only contain letters, numbers, dots, and hyphens"
    end

    unless email.match?(/\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i)
      raise RegistrationError, "Please provide a valid email address"
    end

    unless phone.match?(/\A\+?[1-9]\d{7,14}\z/)
      raise RegistrationError, "Please provide a valid phone number (include country code if international)"
    end

    if usn.present?
      unless usn.match?(/\A[A-Z]{2,3}\d{2}[A-Z]{2}\d{3,4}\z/)
        raise RegistrationError, "Please provide a valid USN format (e.g., 22H51A05P6)"
      end
    end

    if roll_number.present?
      unless roll_number.match?(/\A[\w-]+\z/)
        raise RegistrationError, "Roll number contains invalid characters"
      end
    end

    existing_member = Member.active.find_by(login_id: login_id)
    if existing_member
      raise RegistrationError, "This login ID is already registered"
    end

    existing_member = Member.active.find_by(usn: usn)
    if existing_member
      raise RegistrationError, "This USN is already registered"
    end

    if roll_number.present?
      existing_member = Member.active.find_by(roll_number: roll_number)
      if existing_member
        raise RegistrationError, "This roll number is already registered"
      end
    end

    existing_user = User.find_by(email: email)
    if existing_user && existing_user.member&.active?
      raise RegistrationError, "An account with this email already exists"
    end

    existing_user = User.find_by(phone: phone)
    if existing_user && existing_user.member&.active?
      raise RegistrationError, "An account with this phone number already exists"
    end

    unless %w[1 2 3 4 5 6 7 8].include?(academic_year.to_s)
      raise RegistrationError, "Academic year must be between 1 and 8"
    end
  end
end
