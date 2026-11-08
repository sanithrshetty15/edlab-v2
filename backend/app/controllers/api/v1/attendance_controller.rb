class Api::V1::AttendanceController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :authorize_attendance_access!

  # GET /api/v1/attendance
  def index
    scope = build_attendance_scope

    paginated = paginate(scope.order(entry_date: :desc, entry_time: :desc))

    render json: {
      success: true,
      attendance: paginated.items.map { |a| serialize_attendance(a) },
      pagination: {
        page: paginated.current_page,
        per_page: paginated.per_page,
        total_count: paginated.total_count,
        total_pages: paginated.total_pages
      }
    }, status: :ok
  end

  # GET /api/v1/attendance/:id
  def show
    attendance = AttendanceLog.find(params[:id])

    render json: {
      success: true,
      attendance: serialize_attendance_full(attendance)
    }, status: :ok
  end

  # GET /api/v1/attendance/today
  def today
    today_scope = AttendanceLog.where(entry_date: Date.current)

    if current_user.admin?
      today_scope = filter_attendance_by_params(today_scope)
    else
      member = current_user.member
      today_scope = today_scope.where(member: member) if member
    end

    attendance_records = today_scope.order(entry_time: :desc).limit(50)

    render json: {
      success: true,
      today: attendance_records.map { |a| serialize_attendance(a) },
      summary: today_summary(today_scope)
    }, status: :ok
  end

  # GET /api/v1/attendance/summary
  def summary
    scope = AttendanceLog.all

    if current_user.admin?
      scope = filter_attendance_by_params(scope)
    else
      member = current_user.member
      scope = scope.where(member: member) if member
    end

    date_from = params[:date_from].present? ? Date.parse(params[:date_from]) : 30.days.ago.to_date
    date_to = params[:date_to].present? ? Date.parse(params[:date_to]) : Date.current

    scoped = scope.where(entry_date: date_from..date_to)

    render json: {
      success: true,
      summary: {
        total_login_events: scoped.where(exit_time: nil).count,
        total_logout_events: scoped.where.not(exit_time: nil).count,
        unique_members: scoped.distinct.count(:member_id),
        date_range: { from: date_from, to: date_to }
      }
    }, status: :ok
  end

  # GET /api/v1/attendance/daily-summary
  def daily_summary
    date_from = params[:date_from].present? ? Date.parse(params[:date_from]) : 7.days.ago.to_date
    date_to = params[:date_to].present? ? Date.parse(params[:date_to]) : Date.current

    scope = AttendanceLog.where(entry_date: date_from..date_to)

    if current_user.admin?
      scope = filter_attendance_by_params(scope)
    else
      member = current_user.member
      scope = scope.where(member: member) if member
    end

    summaries = scope.group(:entry_date).select(
      "entry_date",
      "COUNT(*) AS total_events",
      "COUNT(CASE WHEN exit_time IS NULL THEN 1 END) AS logins",
      "COUNT(CASE WHEN exit_time IS NOT NULL THEN 1 END) AS logouts",
      "COUNT(DISTINCT member_id) AS unique_members"
    ).order(entry_date: :desc)

    render json: {
      success: true,
      daily_summary: summaries.map { |s|
        {
          date: s.entry_date,
          total_events: s.total_events,
          logins: s.logins,
          logouts: s.logouts,
          unique_members: s.unique_members
        }
      }
    }, status: :ok
  end

  # GET /api/v1/attendance/occupancy
  def current_occupancy
    raise "Administrator access required" unless current_user.admin?

    active_sessions = AttendanceLog
      .where(entry_date: Date.current)
      .where(exit_time: nil)
      .count

    render json: {
      success: true,
      occupancy: {
        current: active_sessions,
        timestamp: Time.current,
        date: Date.current
      }
    }, status: :ok
  end

  # GET /api/v1/attendance/analytics
  def analytics
    raise "Administrator access required" unless current_user.admin?

    date_from = params[:date_from].present? ? Date.parse(params[:date_from]) : 30.days.ago.to_date
    date_to = params[:date_to].present? ? Date.parse(params[:date_to]) : Date.current

    scope = AttendanceLog.where(entry_date: date_from..date_to)

    filtered_scope = filter_attendance_by_params(scope)

    total_events = filtered_scope.count
    login_events = filtered_scope.where(exit_time: nil).count
    logout_events = filtered_scope.where.not(exit_time: nil).count
    unique_members = filtered_scope.distinct.count(:member_id)
    average_daily_attendance = (filtered_scope.group(:entry_date).count.count.to_f / (date_to - date_from + 1)).round(2)

    branch_stats = filtered_scope
      .joins(:member)
      .group("members.branch")
      .select("members.branch, COUNT(*) AS events, COUNT(DISTINCT attendance_logs.member_id) AS unique_members")
      .order("events DESC")

    year_stats = filtered_scope
      .joins(:member)
      .group("members.academic_year")
      .select("members.academic_year, COUNT(*) AS events, COUNT(DISTINCT attendance_logs.member_id) AS unique_members")
      .order("events DESC")

    render json: {
      success: true,
      analytics: {
        date_range: { from: date_from, to: date_to },
        total_events: total_events,
        login_events: login_events,
        logout_events: logout_events,
        unique_members: unique_members,
        average_daily_attendance: average_daily_attendance,
        by_branch: branch_stats.map { |s| { branch: s.branch, events: s.events, unique_members: s.unique_members } },
        by_academic_year: year_stats.map { |s| { academic_year: s.academic_year, events: s.events, unique_members: s.unique_members } }
      }
    }, status: :ok
  end

  # POST /api/v1/attendance (admin manual entry)
  def create
    raise "Administrator access required" unless current_user.admin?

    result = ManualAttendanceService.record(
      params: attendance_params,
      admin: current_user,
      ip_address: request.remote_ip
    )

    if result[:success]
      AuditLogService.log_attendance_recorded(
        result[:attendance_log],
        source: "manual",
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Attendance recorded successfully",
        attendance: serialize_attendance(result[:attendance_log])
      }, status: :created
    else
      render json: {
        success: false,
        message: result[:error]
      }, status: result[:status] || :unprocessable_entity
    end
  end

  # PATCH /api/v1/attendance/:id/logout
  def record_logout
    raise "Administrator access required" unless current_user.admin?

    attendance = AttendanceLog.find(params[:id])

    unless attendance.exit_time.nil?
      return render json: {
        success: false,
        message: "This attendance record already has a logout time"
      }, status: :unprocessable_entity
    end

    attendance.update!(exit_time: Time.current)

    AuditLogService.log!(
      user: current_user,
      action: "attendance_logout_recorded",
      resource_type: "AttendanceLog",
      resource_id: attendance.id,
      details: {
        member_login_id: attendance.member.login_id,
        member_name: attendance.member.name,
        original_entry_time: attendance.entry_time.to_s,
        logout_time: attendance.exit_time.to_s
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Logout recorded successfully",
      attendance: serialize_attendance(attendance)
    }, status: :ok
  end

  private

  def authorize_attendance_access!
    if current_user.admin?
      # Admins can access all
    elsif current_user.member?
      # Members can only see their own
    else
      render json: {
        success: false,
        message: "Access denied"
      }, status: :forbidden
    end
  end

  def build_attendance_scope
    scope = AttendanceLog.includes(:member, :created_by)

    if current_user.admin?
      scope = filter_attendance_by_params(scope)
    elsif current_user.member?
      member = current_user.member
      scope = scope.where(member: member) if member
    else
      scope = scope.none
    end

    scope
  end

  def filter_attendance_by_params(scope)
    scope = scope.where(entry_date: params[:date]) if params[:date].present?
    scope = scope.where(member_id: params[:member_id]) if params[:member_id].present?
    scope = scope.where("members.academic_year = ?", params[:academic_year].to_i) if params[:academic_year].present?
    scope = scope.joins(:member).where("members.branch = ?", params[:branch]) if params[:branch].present?
    scope = scope.where(source: params[:source]) if params[:source].present?
    scope
  end

  def paginate(scope)
    page = (params[:page] || 1).to_i
    per_page = [[(params[:per_page] || 50).to_i, 200].min, 1].max
    scope.page(page).per(per_page)
  end

  def today_summary(scope)
    {
      total_present: scope.where(exit_time: nil).count,
      total_visited: scope.distinct.count(:member_id),
      date: Date.current
    }
  end

  def serialize_attendance(attendance)
    {
      id: attendance.id,
      member_id: attendance.member_id,
      member: {
        login_id: attendance.member.login_id,
        name: attendance.member.name,
        usn: attendance.member.usn,
        branch: attendance.member.branch
      },
      entry_date: attendance.entry_date.to_s,
      entry_time: attendance.entry_time.iso8601,
      exit_time: attendance.exit_time&.iso8601,
      source: attendance.source,
      device_reference: attendance.device_reference,
      created_by: attendance.created_by&.email,
      created_at: attendance.created_at
    }
  end

  def serialize_attendance_full(attendance)
    serialize_attendance(attendance).merge(
      updated_at: attendance.updated_at
    )
  end

  def attendance_params
    params.permit(:member_id, :entry_date, :entry_time, :source, :device_reference)
  end
end
