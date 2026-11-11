class Api::V1::EventsController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :require_admin!, only: [:create, :update, :destroy]

  # GET /api/v1/events
  def index
    scope = Event
      .where(deleted_at: nil)
      .order(date: :desc, start_time: :desc)

    if params[:upcoming].present? && params[:upcoming] == "true"
      scope = scope.where("date >= ?", Date.current)
    end

    if params[:past].present? && params[:past] == "true"
      scope = scope.where("date < ?", Date.current)
    end

    if params[:date].present?
      scope = scope.where(date: Date.parse(params[:date]))
    end

    if params[:event_type].present?
      scope = scope.where(event_type: params[:event_type])
    end

    if params[:before].present?
      scope = scope.where("date < ?", Date.parse(params[:before]))
    end

    if params[:after].present?
      scope = scope.where("date > ?", Date.parse(params[:after]))
    end

    events = paginate(scope.includes(:created_by))

    render json: {
      success: true,
      events: events.items.map { |e| serialize_event(e) },
      pagination: {
        page: events.current_page,
        per_page: events.per_page,
        total_count: events.total_count
      }
    }, status: :ok
  end

  # GET /api/v1/events/:id
  def show
    event = Event.find_by(id: params[:id], deleted_at: nil)

    unless event
      return render json: {
        success: false,
        message: "Event not found"
      }, status: :not_found
    end

    render json: {
      success: true,
      event: serialize_event_full(event)
    }, status: :ok
  end

  # POST /api/v1/events
  def create
    title = params[:title].to_s.strip
    description = params[:description].to_s.strip
    event_type = params[:event_type].to_s.downcase
    date = params[:date].present? ? Date.parse(params[:date]) : nil
    start_time = params[:start_time].present? ? Time.parse(params[:start_time]) : nil
    end_time = params[:end_time].present? ? Time.parse(params[:end_time]) : nil
    venue = params[:venue].to_s.strip
    external_links = params[:external_links] || []

    validate_event_inputs(
      title: title,
      description: description,
      event_type: event_type,
      date: date,
      start_time: start_time,
      end_time: end_time
    )

    event = Event.create!(
      title: title,
      description: description.presence,
      event_type: event_type,
      date: date,
      start_time: start_time || Time.current.change(hour: 10, min: 0),
      end_time: end_time || Time.current.change(hour: 18, min: 0),
      venue: venue.presence,
      external_links: Array(external_links).map(&:to_s).uniq,
      created_by: current_user
    )

    AuditLogService.log_event_created(event, created_by: current_user)

    NotificationService.notify_event(event, created_by: current_user)

    render json: {
      success: true,
      message: "Event created successfully",
      event: serialize_event_full(event)
    }, status: :created
  end

  # PATCH /api/v1/events/:id
  def update
    event = Event.find_by(id: params[:id], deleted_at: nil)

    unless event
      return render json: {
        success: false,
        message: "Event not found"
      }, status: :not_found
    end

    changes = {}

    if params[:title].present? && event.title != params[:title].to_s.strip
      title = params[:title].to_s.strip
      unless title.present?
        return render json: {
          success: false,
          message: "Title cannot be empty"
        }, status: :unprocessable_entity
      end
      changes[:title] = { from: event.title, to: title }
      event.update!(title: title)
    end

    if params[:description].present? && event.description != params[:description].to_s.strip
      changes[:description] = { from: event.description.truncate(100), to: params[:description].truncate(100) }
      event.update!(description: params[:description].to_s.strip.presence)
    end

    if params[:event_type].present? && event.event_type != params[:event_type].to_s.downcase
      changes[:event_type] = { from: event.event_type, to: params[:event_type] }
      event.update!(event_type: params[:event_type].to_s.downcase)
    end

    if params[:date].present?
      date = Date.parse(params[:date])
      if event.date != date
        changes[:date] = { from: event.date.to_s, to: date.to_s }
        event.update!(date: date)
      end
    end

    if params[:start_time].present?
      start_time = Time.parse(params[:start_time])
      if event.start_time != start_time
        changes[:start_time] = { from: event.start_time.to_s, to: start_time.to_s }
        event.update!(start_time: start_time)
      end
    end

    if params[:end_time].present?
      end_time = Time.parse(params[:end_time])
      if event.end_time != end_time
        changes[:end_time] = { from: event.end_time.to_s, to: end_time.to_s }
        event.update!(end_time: end_time)
      end
    end

    if params[:venue].present? && event.venue != params[:venue].to_s.strip
      changes[:venue] = { from: event.venue, to: params[:venue] }
      event.update!(venue: params[:venue].to_s.strip.presence)
    end

    if params[:external_links].present?
      links = Array(params[:external_links]).map(&:to_s).uniq
      if event.external_links != links
        changes[:external_links] = { from: event.external_links, to: links }
        event.update!(external_links: links)
      end
    end

    if changes.any?
      AuditLogService.log!(
        user: current_user,
        action: "event_updated",
        resource_type: "Event",
        resource_id: event.id,
        details: changes,
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Event updated",
        event: serialize_event_full(event)
      }, status: :ok
    else
      render json: {
        success: false,
        message: "No changes provided"
      }, status: :unprocessable_entity
    end
  end

  # DELETE /api/v1/events/:id
  def destroy
    event = Event.find_by(id: params[:id], deleted_at: nil)

    unless event
      return render json: {
        success: false,
        message: "Event not found"
      }, status: :not_found
    end

    event.update!(deleted_at: Time.current)

    AuditLogService.log!(
      user: current_user,
      action: "event_deleted",
      resource_type: "Event",
      resource_id: event.id,
      details: { title: event.title, event_type: event.event_type },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Event deleted"
    }, status: :ok
  end

  private

  def require_admin!
    unless current_user.admin?
      return render json: {
        success: false,
        message: "Administrator access required"
      }, status: :forbidden
    end
  end

  def validate_event_inputs(title:, description:, event_type:, date:, start_time:, end_time:)
    raise "Title is required" if title.blank?
    raise "Event type is required" if event_type.blank?

    unless %w[workshop seminar talk hackathon training other].include?(event_type)
      raise "Invalid event type"
    end

    raise "Date is required" if date.blank?
    raise "Start time is required" if start_time.blank?
    raise "End time is required" if end_time.blank?

    raise "End time must be after start time" if end_time <= start_time
  end


  def serialize_event(event)
    {
      id: event.id,
      title: event.title,
      event_type: event.event_type,
      date: event.date.to_s,
      start_time: event.start_time.strftime("%H:%M"),
      end_time: event.end_time.strftime("%H:%M"),
      venue: event.venue,
      created_by: {
        id: event.created_by.id,
        name: event.created_by.admin&.name || event.created_by.email
      },
      created_at: event.created_at
    }
  end

  def serialize_event_full(event)
    serialize_event(event).merge(
      description: event.description,
      external_links: event.external_links,
      is_upcoming: event.date >= Date.current,
      ends_at: event.end_time.iso8601
    )
  end
end
