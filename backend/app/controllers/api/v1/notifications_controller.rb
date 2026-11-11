class Api::V1::NotificationsController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!

  # GET /api/v1/notifications
  def index
    scope = Notification.where(user_id: current_user.id)
      .order(created_at: :desc)

    if params[:type].present?
      scope = scope.where(notification_type: params[:type])
    end

    if params[:read].present?
      scope = scope.where(is_read: params[:read] == "true")
    end

    if params[:before].present?
      scope = scope.where("created_at < ?", Time.parse(params[:before]))
    end

    notifications = paginate(scope)

    render json: {
      success: true,
      notifications: notifications.items.map { |n| serialize_notification(n) },
      pagination: {
        page: notifications.current_page,
        per_page: notifications.per_page,
        total_count: notifications.total_count,
        total_pages: notifications.total_pages,
        unread_count: Notification.where(user_id: current_user.id, is_read: false).count
      }
    }, status: :ok
  end

  # GET /api/v1/notifications/unread-count
  def unread_count
    count = Notification.where(user_id: current_user.id, is_read: false).count

    render json: {
      success: true,
      unread_count: count
    }, status: :ok
  end

  # GET /api/v1/notifications/:id
  def show
    notification = Notification.find_by(id: params[:id], user_id: current_user.id)

    unless notification
      return render json: {
        success: false,
        message: "Notification not found"
      }, status: :not_found
    end

    render json: {
      success: true,
      notification: serialize_notification(notification)
    }, status: :ok
  end

  # POST /api/v1/notifications/read
  def mark_read
    ids = params[:ids] || []

    if ids.blank?
      Notification.where(user_id: current_user.id, is_read: false)
        .update_all(is_read: true, read_at: Time.current)
    else
      Notification.where(id: ids, user_id: current_user.id, is_read: false)
        .update_all(is_read: true, read_at: Time.current)
    end

    render json: {
      success: true,
      message: "Notifications marked as read"
    }, status: :ok
  end

  # POST /api/v1/notifications/read-all
  def mark_all_read
    Notification.where(user_id: current_user.id, is_read: false)
      .update_all(is_read: true, read_at: Time.current)

    render json: {
      success: true,
      message: "All notifications marked as read"
    }, status: :ok
  end

  # DELETE /api/v1/notifications/:id
  def destroy
    notification = Notification.find_by(id: params[:id], user_id: current_user.id)

    unless notification
      return render json: {
        success: false,
        message: "Notification not found"
      }, status: :not_found
    end

    notification.destroy!

    render json: {
      success: true,
      message: "Notification deleted"
    }, status: :ok
  end

  # GET /api/v1/notifications/history
  def history
    scope = Notification.where(user_id: current_user.id)
      .order(created_at: :desc)

    date_from = params[:date_from].present? ? Date.parse(params[:date_from]) : 30.days.ago.to_date
    date_to = params[:date_to].present? ? Date.parse(params[:date_to]) : Date.current

    scope = scope.where(created_at: date_from.beginning_of_day..date_to.end_of_day)

    notifications = paginate(scope)

    grouped = notifications.items.group_by { |n| n.created_at.to_date }

    render json: {
      success: true,
      history: grouped.map { |date, notifications|
        {
          date: date.to_s,
          count: notifications.count,
          notifications: notifications.map { |n| serialize_notification(n) }
        }
      },
      pagination: {
        page: notifications.current_page,
        total_count: notifications.total_count
      }
    }, status: :ok
  end

  private


  def serialize_notification(notification)
    {
      id: notification.id,
      title: notification.title,
      body: notification.body,
      notification_type: notification.notification_type,
      subject_type: notification.subject_type,
      subject_id: notification.subject_id,
      is_read: notification.is_read,
      read_at: notification.read_at,
      created_at: notification.created_at
    }
  end
end
