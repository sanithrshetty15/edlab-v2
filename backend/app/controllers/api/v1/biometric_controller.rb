class Api::V1::BiometricController < Api::V1::BaseController
  skip_before_action :authenticate_user!, only: [:authenticate_device, :submit_event, :sync_events]
  before_action :authenticate_device!, only: [:enroll_fingerprint, :remove_enrollment, :device_status]
  before_action :authenticate_user!, only: [:device_status]
  before_action :require_admin!, only: [:enroll_fingerprint, :remove_enrollment]

  # POST /api/v1/biometric/authenticate-device
  def authenticate_device
    api_key = params[:api_key].to_s
    device_name = params[:device_name].to_s

    unless api_key.present?
      return render json: {
        success: false,
        message: "API key is required"
      }, status: :unauthorized
    end

    device = DeviceRegistration.find_by(api_key_digest: Digest::SHA256.hexdigest(api_key))

    unless device
      return render json: {
        success: false,
        message: "Invalid device credentials"
      }, status: :unauthorized
    end

    unless device.status == "active"
      return render json: {
        success: false,
        message: "Device is not active"
      }, status: :forbidden
    end

    device.update!(
      last_seen_at: Time.current,
      name: device_name.presence || device.name
    )

    device_token = SecureRandom.hex(32)

    render json: {
      success: true,
      message: "Device authenticated successfully",
      device: {
        id: device.id,
        name: device.name,
        device_type: device.device_type,
        status: device.status,
        token: device_token,
        expires_in: 3600
      }
    }, status: :ok
  end

  # POST /api/v1/biometric/enroll-fingerprint
  def enroll_fingerprint
    device_registration = @current_device
    member_id = params[:member_id].to_i
    fingerprint_template = params[:fingerprint_template].to_s
    device_user_id = params[:device_user_id].to_s

    unless fingerprint_template.present?
      return render json: {
        success: false,
        message: "Fingerprint template is required"
      }, status: :unprocessable_entity
    end

    member = Member.active.find_by(id: member_id)
    unless member
      return render json: {
        success: false,
        message: "Member not found or inactive"
      }, status: :not_found
    end

    existing_enrollment = BiometricEnrollment.find_by(
      member_id: member_id,
      device_registration_id: device_registration.id
    )

    if existing_enrollment
      existing_enrollment.update!(
        device_user_id: device_user_id.presence || existing_enrollment.device_user_id,
        status: "enrolled",
        updated_at: Time.current
      )
    else
      existing_enrollment = BiometricEnrollment.create!(
        member_id: member_id,
        device_registration_id: device_registration.id,
        device_user_id: device_user_id.presence || SecureRandom.hex(8),
        status: "enrolled"
      )
    end

    AuditLogService.log_biometric_enrollment_confirmed(
      existing_enrollment,
      confirmed_by: current_user,
      device_user_id: existing_enrollment.device_user_id,
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Fingerprint enrolled successfully",
      enrollment: {
        id: existing_enrollment.id,
        member_id: existing_enrollment.member_id,
        member_name: existing_enrollment.member.name,
        member_login_id: existing_enrollment.member.login_id,
        device_user_id: existing_enrollment.device_user_id,
        status: existing_enrollment.status
      }
    }, status: :ok
  end

  # DELETE /api/v1/biometric/enrollments/:id
  def remove_enrollment
    enrollment = BiometricEnrollment.find(params[:id])

    unless enrollment.device_registration_id == @current_device.id
      return render json: {
        success: false,
        message: "This enrollment does not belong to this device"
      }, status: :forbidden
    end

    member = enrollment.member

    AuditLogService.log_biometric_enrollment_removed(
      enrollment,
      removed_by: current_user,
      ip_address: request.remote_ip
    )

    enrollment.destroy!

    render json: {
      success: true,
      message: "Fingerprint enrollment removed successfully",
      member: {
        id: member.id,
        login_id: member.login_id,
        name: member.name
      }
    }, status: :ok
  end

  # GET /api/v1/biometric/enrollments
  def list_enrollments
    enrollments = BiometricEnrollment
      .includes(:member, :device_registration)
      .where(device_registration_id: @current_device.id)
      .order(updated_at: :desc)

    render json: {
      success: true,
      enrollments: enrollments.map { |e|
        {
          id: e.id,
          member_id: e.member_id,
          member_name: e.member.name,
          member_login_id: e.member.login_id,
          member_usn: e.member.usn,
          device_user_id: e.device_user_id,
          status: e.status,
          enrolled_at: e.created_at
        }
      }
    }, status: :ok
  end

  # GET /api/v1/biometric/status
  def device_status
    device = @current_device

    pending_events = DeviceEvent
      .where(device_registration_id: device.id, processing_status: "pending")
      .count

    render json: {
      success: true,
      device: {
        id: device.id,
        name: device.name,
        device_type: device.device_type,
        status: device.status,
        last_seen_at: device.last_seen_at,
        pending_events: pending_events
      }
    }, status: :ok
  end

  # POST /api/v1/biometric/submit-event
  def submit_event
    device_registration_id = params[:device_id].to_i
    event_type = params[:event_type].to_s
    idempotency_key = params[:idempotency_key].to_s
    member_identifier = params[:member_identifier].to_s
    event_timestamp = params[:event_timestamp].to_s
    hmac_signature = params[:hmac_signature].to_s
    payload = params[:payload] || {}

    unless device_registration_id.present? && event_type.present? && idempotency_key.present?
      return render json: {
        success: false,
        message: "Device ID, event type, and idempotency key are required"
      }, status: :unprocessable_entity
    end

    device = DeviceRegistration.find_by(id: device_registration_id)
    unless device
      return render json: {
        success: false,
        message: "Device not found"
      }, status: :not_found
    end

    unless device.status == "active"
      return render json: {
        success: false,
        message: "Device is not active"
      }, status: :forbidden
    end

    unless %w[fingerprint_scan finger_mismatch enrollment_success enrollment_failure].include?(event_type)
      return render json: {
        success: false,
        message: "Invalid event type"
      }, status: :unprocessable_entity
    end

    existing_event = DeviceEvent.find_by(idempotency_key: idempotency_key)
    if existing_event
      return render json: {
        success: true,
        message: "Event already processed",
        event: serialize_device_event(existing_event)
      }, status: :ok
    end

    device_event = DeviceEvent.create!(
      device_registration_id: device_registration_id,
      event_type: event_type,
      idempotency_key: idempotency_key,
      member_identifier: member_identifier.presence,
      hmac_signature: hmac_signature.presence,
      event_timestamp: parse_event_timestamp(event_timestamp),
      payload: payload.with_indifferent_access
    )

    device.update!(last_seen_at: Time.current)

    EventProcessingJob.perform_later(device_event.id)

    render json: {
      success: true,
      message: "Event received and queued for processing",
      event: serialize_device_event(device_event)
    }, status: :accepted
  end

  # POST /api/v1/biometric/sync-events
  def sync_events
    device_registration_id = params[:device_id].to_i
    last_sync_timestamp = params[:last_sync_timestamp].to_s

    unless device_registration_id.present?
      return render json: {
        success: false,
        message: "Device ID is required"
      }, status: :unprocessable_entity
    end

    device = DeviceRegistration.find_by(id: device_registration_id)
    unless device
      return render json: {
        success: false,
        message: "Device not found"
      }, status: :not_found
    end

    scope = DeviceEvent.where(device_registration_id: device_registration_id)

    if last_sync_timestamp.present?
      last_sync = Time.parse(last_sync_timestamp) rescue nil
      scope = scope.where("created_at > ?", last_sync) if last_sync
    end

    events = scope.order(created_at: :asc).limit(100)

    render json: {
      success: true,
      events: events.map { |e| serialize_device_event(e) },
      sync_metadata: {
        last_sync_timestamp: Time.current.iso8601,
        events_synced: events.count,
        has_more: events.count >= 100
      }
    }, status: :ok
  end

  private

  def authenticate_device!
    device_token = request.headers["X-Device-Token"].to_s

    unless device_token.present?
      return render json: {
        success: false,
        message: "Device token is required"
      }, status: :unauthorized
    end

    @current_device = DeviceRegistration.find_by(
      api_key_digest: Digest::SHA256.hexdigest(device_token)
    )

    unless @current_device && @current_device.status == "active"
      return render json: {
        success: false,
        message: "Invalid or inactive device"
      }, status: :unauthorized
    end

    @current_device.update!(last_seen_at: Time.current)
  end

  def require_admin!
    authenticate_user!
    unless current_user.admin?
      return render json: {
        success: false,
        message: "Administrator access required"
      }, status: :forbidden
    end
  end

  def parse_event_timestamp(timestamp_str)
    return Time.current if timestamp_str.blank?

    begin
      parsed = Time.parse(timestamp_str)
      [
        parsed,
        Time.current - 5.minutes,
        Time.current + 5.minutes
      ].max
    rescue ArgumentError
      Time.current
    end
  end

  def serialize_device_event(event)
    {
      id: event.id,
      device_registration_id: event.device_registration_id,
      event_type: event.event_type,
      idempotency_key: event.idempotency_key,
      member_identifier: event.member_identifier,
      event_timestamp: event.event_timestamp.iso8601,
      payload: event.payload,
      processing_status: event.processing_status,
      created_at: event.created_at
    }
  end
end
