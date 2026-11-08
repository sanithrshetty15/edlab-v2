module PhoneValidation
  extend ActiveSupport::Concern

  included do
    validates :phone, presence: true
    validates :phone, format: { with: /\\A\\+?[1-9]\\d{7,14}\\z/, message: "must be a valid phone number (include country code if international)" }, allow_blank: true
  end

  def phone_phone_verified?
    phone.present? && phone_verified?
  end

  def verify_phone!(code)
    target_phone = respond_to?(:phone) ? phone : nil
    target_phone ||= user.phone if respond_to?(:user) && user
    return false unless target_phone.present?

    TwilioService.check_verification(phone_number: target_phone, code: code)
  rescue StandardError
    false
  end
end
