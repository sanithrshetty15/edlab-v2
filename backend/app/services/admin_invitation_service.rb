class AdminInvitationService
  def self.invite(params:, invited_by:, ip_address: nil)
    email = params[:email].to_s.strip.downcase
    name = params[:name].to_s.strip
    phone = params[:phone].to_s.strip
    academic_year = params[:academic_year].to_i
    branch = params[:branch].to_s.strip
    section = params[:section].to_s.strip
    usn = params[:usn].to_s.strip.upcase
    roll_number = params[:roll_number].to_s.strip
    password = params[:password].to_s

    validate_invitation(
      email: email,
      name: name,
      phone: phone,
      academic_year: academic_year,
      branch: branch,
      password: password
    )

    existing_member = Member.active.find_by(email: email)
    if existing_member
      raise "A member with this email already exists"
    end

    existing_user = User.find_by(email: email)
    if existing_user
      raise "An account with this email already exists"
    end

    if Admin.count >= 3
      raise "Maximum of 3 administrators allowed"
    end

    if params[:is_primary].to_s == "true" && Admin.primary.count >= 1
      raise "Cannot invite another primary admin"
    end

    login_id = generate_login_id(email, name)

    if Member.active.find_by(login_id: login_id)
      login_id = "#{login_id}_#{SecureRandom.hex(4)}"
    end

    ActiveRecord::Base.transaction do
      user = User.create!(
        email: email,
        phone: phone.presence,
        password: password,
        password_confirmation: password,
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

      if params[:accept_rules].to_s == "true"
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
        user: user,
        member_data: {
          email: email,
          name: name,
          login_id: login_id,
          academic_year: academic_year,
          branch: branch,
          section: section.presence
        }
      }
    end
  rescue => e
    Rails.logger.error("Admin invitation failed: #{e.class} - #{e.message}")
    {
      success: false,
      error: e.message,
      status: :unprocessable_entity
    }
  end

  def self.validate_invitation(email:, name:, phone:, academic_year:, branch:, password:)
    raise "Name is required" if name.blank?
    raise "Email is required" if email.blank?
    raise "Password is required" if password.blank?
    raise "Password must be at least 8 characters" if password.length < 8
    raise "Academic year is required" if academic_year.blank?
    raise "Branch is required" if branch.blank?

    unless email.match?(/\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i)
      raise "Please provide a valid email address"
    end

    unless phone.blank? || phone.match?(/\A\+?[1-9]\d{7,14}\z/)
      raise "Please provide a valid phone number"
    end

    unless %w[1 2 3 4 5 6 7 8].include?(academic_year.to_s)
      raise "Academic year must be between 1 and 8"
    end
  end

  def self.generate_login_id(email, name)
    base = name.to_s.downcase.gsub(/[^a-z0-9]/, "").first(8)
    domain = email.to_s.split("@").first.to_s.downcase.gsub(/[^a-z0-9]/, "").first(6)
    "#{base}_#{domain}"
  end
end
