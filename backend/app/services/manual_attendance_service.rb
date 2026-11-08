class ManualAttendanceService
  class AttendanceError < StandardError; end

  def self.record(params:, admin:, ip_address: nil)
    member_id = params[:member_id].to_i
    entry_date = params[:entry_date].present? ? Date.parse(params[:entry_date]) : Date.current
    entry_time = params[:entry_time].present? ? Time.parse(params[:entry_time]) : Time.current
    source = params[:source].to_s.presence || "manual"
    device_reference = params[:device_reference].to_s.presence

    member = Member.active.find_by(id: member_id)
    unless member
      return { success: false, error: "Member not found or inactive", status: :not_found }
    end

    if params[:entry_date].blank? && params[:entry_time].blank?
      existing_today = AttendanceLog
        .where(member_id: member_id)
        .where(entry_date: Date.current)
        .where(exit_time: nil)
        .first

      if existing_today
        return {
          success: false,
          error: "Member already has an active attendance record for today",
          status: :conflict
        }
      end
    end

    if entry_time.to_date != entry_date
      return {
        success: false,
        error: "Entry time date must match entry date",
        status: :unprocessable_entity
      }
    end

    if entry_time < entry_date.beginning_of_day
      return {
        success: false,
        error: "Entry time cannot be before the entry date",
        status: :unprocessable_entity
      }
    end

    ActiveRecord::Base.transaction do
      attendance_log = AttendanceLog.create!(
        member_id: member_id,
        entry_date: entry_date,
        entry_time: entry_time,
        source: source,
        device_reference: device_reference,
        created_by: admin
      )

      {
        success: true,
        attendance_log: attendance_log
      }
    end
  rescue Date::Error, ArgumentError
    {
      success: false,
      error: "Invalid date or time format",
      status: :unprocessable_entity
    }
  rescue ActiveRecord::RecordInvalid => e
    {
      success: false,
      error: e.message,
      status: :unprocessable_entity
    }
  rescue => e
    Rails.logger.error("Manual attendance recording failed: #{e.class} - #{e.message}")
    {
      success: false,
      error: "Failed to record attendance",
      status: :internal_server_error
    }
  end
end
