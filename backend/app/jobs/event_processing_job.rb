class EventProcessingJob < ApplicationJob
  queue_as :default

  def perform(device_event_id)
    device_event = DeviceEvent.find_by(id: device_event_id)
    return unless device_event

    begin
      process_device_event(device_event)
    rescue => e
      Rails.logger.error("Event processing failed: #{e.class} - #{e.message}")
      device_event.update!(
        processing_status: "failed",
        processing_error: "#{e.class}: #{e.message}"
      )
    end
  end

  private

  def process_device_event(device_event)
    device = device_event.device_registration
    event_type = device_event.event_type

    case event_type
    when "fingerprint_scan"
      process_fingerprint_scan(device_event, device)
    when "finger_mismatch"
      process_finger_mismatch(device_event, device)
    when "enrollment_success", "enrollment_failure"
      process_enrollment_event(device_event, device)
    end

    device_event.update!(
      processing_status: "completed"
    )
  end

  def process_fingerprint_scan(device_event, device)
    member_identifier = device_event.member_identifier
    return unless member_identifier.present?

    member = Member.active.find_by(login_id: member_identifier)

    if member.nil?
      Rails.logger.warn(
        "Biometric scan: unknown member identifier '#{member_identifier}' from device #{device.id}"
      )
      return
    end

    last_attendance = AttendanceLog
      .where(member_id: member.id)
      .where(entry_date: Date.current)
      .order(entry_time: :desc)
      .first

    if last_attendance.nil? || last_attendance.exit_time.present?
      action = "login"
      server_time = Time.current

      AttendanceLog.create!(
        member_id: member.id,
        entry_date: server_time.to_date,
        entry_time: server_time,
        source: "biometric",
        device_reference: device_event.id.to_s,
        created_by: device.user
      )

      AuditLogService.log!(
        user: device.user,
        action: "attendance_login_biometric",
        resource_type: "AttendanceLog",
        resource_id: nil,
        details: {
          member_login_id: member.login_id,
          member_name: member.name,
          device_id: device.id,
          server_time: server_time.iso8601
        },
        ip_address: nil
      )
    else
      server_time = Time.current

      last_attendance.update!(
        exit_time: server_time
      )

      AuditLogService.log!(
        user: device.user,
        action: "attendance_logout_biometric",
        resource_type: "AttendanceLog",
        resource_id: last_attendance.id,
        details: {
          member_login_id: member.login_id,
          member_name: member.name,
          device_id: device.id,
          logout_time: server_time.iso8601
        },
        ip_address: nil
      )
    end
  end

  def process_finger_mismatch(device_event, device)
    Rails.logger.info(
      "Biometric mismatch event from device #{device.id}: #{device_event.member_identifier}"
    )
  end

  def process_enrollment_event(device_event, device)
    Rails.logger.info(
      "Enrollment event from device #{device.id}: #{device_event.event_type}"
    )
  end
end
