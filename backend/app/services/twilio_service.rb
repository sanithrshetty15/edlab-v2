require "net/http"
require "json"
require "uri"

class TwilioService
  VERIFY_BASE_URL = "https://verify.twilio.com/v2"

  def self.send_verification(phone_number:, channel: "sms")
    account_sid = ENV["TWILIO_ACCOUNT_SID"]
    auth_token = ENV["TWILIO_AUTH_TOKEN"]
    service_sid = ENV["TWILIO_VERIFY_SERVICE_SID"]

    if account_sid.blank? || auth_token.blank? || service_sid.blank?
      raise "Twilio Verify credentials are not configured"
    end

    unless phone_number.to_s.match?(/\A\+[1-9]\d{7,14}\z/)
      raise ArgumentError, "Invalid phone number format"
    end

    uri = URI("#{VERIFY_BASE_URL}/Services/#{service_sid}/Verifications")
    result = post_form(uri, account_sid, auth_token, "To" => phone_number.to_s, "Channel" => channel)

    status = result["status"].to_s
    unless %w[pending approved].include?(status) || result["sid"].present?
      raise "Twilio Verify could not send the verification code"
    end

    result
  rescue JSON::ParserError
    raise "Twilio returned an invalid response"
  rescue Net::OpenTimeout, Net::ReadTimeout
    raise "Twilio request timed out"
  rescue SocketError, SystemCallError
    raise "Unable to connect to Twilio"
  end

  def self.check_verification(phone_number:, code:)
    account_sid = ENV["TWILIO_ACCOUNT_SID"]
    auth_token = ENV["TWILIO_AUTH_TOKEN"]
    service_sid = ENV["TWILIO_VERIFY_SERVICE_SID"]

    if account_sid.blank? || auth_token.blank? || service_sid.blank?
      raise "Twilio Verify credentials are not configured"
    end

    unless phone_number.to_s.match?(/\A\+[1-9]\d{7,14}\z/)
      raise ArgumentError, "Invalid phone number format"
    end

    code = code.to_s.strip
    raise ArgumentError, "Invalid verification code format" unless code.match?(/\A\d{4,10}\z/)

    uri = URI("#{VERIFY_BASE_URL}/Services/#{service_sid}/VerificationCheck")
    result = post_form(uri, account_sid, auth_token, "To" => phone_number.to_s, "Code" => code)

    result["status"].to_s == "approved"
  rescue JSON::ParserError
    raise "Twilio returned an invalid response"
  rescue Net::OpenTimeout, Net::ReadTimeout
    raise "Twilio request timed out"
  rescue SocketError, SystemCallError
    raise "Unable to connect to Twilio"
  end

  def self.post_form(uri, account_sid, auth_token, form_data)
    request = Net::HTTP::Post.new(uri)
    request.basic_auth(account_sid, auth_token)
    request["Content-Type"] = "application/x-www-form-urlencoded"
    request.set_form_data(form_data)

    response = Net::HTTP.start(
      uri.hostname,
      uri.port,
      use_ssl: true,
      open_timeout: 5,
      read_timeout: 15
    ) do |http|
      http.request(request)
    end

    result = parse_response_body(response)

    unless response.is_a?(Net::HTTPSuccess)
      raise twilio_error_message(response, result)
    end

    result
  end
  private_class_method :post_form

  def self.parse_response_body(response)
    body = response.body.to_s
    return {} if body.empty?

    JSON.parse(body)
  end
  private_class_method :parse_response_body

  def self.twilio_error_message(response, result)
    # Twilio Verify returns "status" => "pending"/"approved" on success; on check failure
    # it may return 200 with status != approved, handled by caller. For HTTP errors:
    if result.is_a?(Hash)
      code = result["code"]
      message = result["message"]
      status = result["status"].to_s

      # VerificationCheck: incorrect/expired codes can be represented as non-approved.
      # HTTP 404 with status not found is handled as an error; caller can treat as failure.
      if message.present?
        # Do not leak sensitive details; provide safe, generic mapping for common cases.
        if code.to_s == "20404" || message.match?(/not found/i)
          return "Twilio Verify could not find a matching verification request"
        end
        if message.match?(/expired/i)
          return "Verification code has expired"
        end
        if message.match?(/invalid/i)
          return "Invalid verification code"
        end
        return "Twilio Verify API error"
      end
    end

    "Twilio Verify API error: HTTP #{response.code}"
  end
  private_class_method :twilio_error_message
end
