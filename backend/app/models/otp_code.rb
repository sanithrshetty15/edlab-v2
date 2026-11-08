class OtpCode < ApplicationRecord
  belongs_to :user

    validates :otp_digest, :delivery_method, :purpose, :expires_at, presence: true

  validates :attempts, numericality: {
    only_integer: true,
    greater_than_or_equal_to: 0,
    less_than_or_equal_to: 5
  }

  validates :delivery_method, inclusion: {
    in: %w[email sms]
  }

  validates :purpose, inclusion: {
    in: %w[password_reset first_password_change phone_verification contact_email_change]
  }

  scope :unused, -> { where(used: false) }
  scope :unexpired, -> { where("expires_at > ?", Time.current) }

  def expired?
    expires_at <= Time.current
  end

  def exhausted?
    attempts >= 5
  end

  def self.generate_code
    SecureRandom.random_number(10_000).to_s.rjust(4, "0")
  end

  def self.digest_code(code)
    OpenSSL::HMAC.hexdigest(
     "SHA256",
     Rails.application.secret_key_base,
     code
    )
  end

  def self.issue_for(user, delivery_method:, purpose:)
  recent_count = where(user: user, purpose: purpose)
    .where("created_at > ?", 15.minutes.ago)
    .count

  raise "OTP request limit exceeded" if recent_count >= 3

  transaction do
    where(user: user, purpose: purpose, used: false)
      .update_all(used: true)

    code = generate_code

    otp = user.otp_codes.create!(
      otp_digest: digest_code(code),
      delivery_method: delivery_method,
      purpose: purpose,
      attempts: 0,
      used: false,
      expires_at: 5.minutes.from_now
    )

    [otp, code]
  end
end

  # Tracks an SMS verification request handled by Twilio Verify (no Twilio-generated
  # OTP code is stored). Preserves request rate limiting and invalidation semantics.
  def self.track_verify_request_for!(user, purpose:)
    recent_count = where(user: user, purpose: purpose)
      .where("created_at > ?", 15.minutes.ago)
      .count

    raise "OTP request limit exceeded" if recent_count >= 3

    transaction do
      where(user: user, purpose: purpose, used: false)
        .update_all(used: true)

      user.otp_codes.create!(
        otp_digest: digest_code(SecureRandom.hex(32)),
        delivery_method: "sms",
        purpose: purpose,
        attempts: 0,
        used: false,
        expires_at: 10.minutes.from_now
      )
    end
  end

  def verify_code(code)
    return false unless code.to_s.match?(/\A\d{4}\z/)
    with_lock do
     return false if used?
     return false if expired?
     return false if exhausted?

      increment!(:attempts)

      valid = ActiveSupport::SecurityUtils.secure_compare(
       otp_digest,
        self.class.digest_code(code.to_s)
      )

      update!(used: true) if valid

     valid
    end
  end
end

