class OtpDeliveryService
  def self.send_email(user:, purpose:)
    otp, code = OtpCode.issue_for(
      user,
      delivery_method: "email",
      purpose: purpose
    )

    OtpMailer.verification_email(user, code).deliver_now

    {
      success: true,
      message: "Verification code sent successfully",
      expires_in: 300
    }
  rescue StandardError => e
    otp&.update(used: true) if otp&.persisted?

    Rails.logger.error(
      "OTP delivery failed: #{e.class} - #{e.message}"
    )

    {
      success: false,
      message: "Unable to send verification code"
    }
  end

  def self.send_sms(user:, purpose:)
    member = user.member
    phone = member&.phone

    unless phone.present?
      raise "No phone number available for SMS delivery"
    end

    otp = OtpCode.track_verify_request_for!(user, purpose)

    TwilioService.send_verification(
      phone_number: phone,
      channel: "sms"
    )

    {
      success: true,
      message: "Verification code sent successfully",
      expires_in: 300
    }
  rescue StandardError => e
    otp&.update(used: true) if otp&.respond_to?(:persisted?) && otp.persisted?

    Rails.logger.error(
      "SMS OTP delivery failed: #{e.class} - #{e.message}"
    )

    {
      success: false,
      message: "Unable to send verification code"
    }
  end
end