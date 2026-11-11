class Api::V1::AnnouncementsController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_admin!, only: [:create, :update, :destroy, :publish, :toggle_pin]

  # GET /api/v1/announcements
  def index
    scope = Announcement
      .where(deleted_at: nil)
      .order(published_at: :desc, created_at: :desc)

    if params[:published].present? && params[:published] == "true"
      scope = scope.where.not(published_at: nil)
    end

    if params[:important].present? && params[:important] == "true"
      scope = scope.where(is_pinned: true)
    end

    if params[:before].present?
      scope = scope.where("published_at < ? OR (published_at IS NULL AND created_at < ?)", Time.parse(params[:before]), Time.parse(params[:before]))
    end

    announcements = paginate(scope.includes(:created_by))

    render json: {
      success: true,
      announcements: announcements.items.map { |a| serialize_announcement(a) },
      pagination: {
        page: announcements.current_page,
        per_page: announcements.per_page,
        total_count: announcements.total_count,
        total_pages: announcements.total_pages
      }
    }, status: :ok
  end

  # GET /api/v1/announcements/:id
  def show
    announcement = Announcement.find_by(id: params[:id], deleted_at: nil)

    unless announcement
      return render json: {
        success: false,
        message: "Announcement not found"
      }, status: :not_found
    end

    unless announcement.published_at.present? || current_user.admin?
      return render json: {
        success: false,
        message: "Announcement is not yet published"
      }, status: :forbidden
    end

    user_read = announcement.announcement_reads.find_by(user_id: current_user.id)

    render json: {
      success: true,
      announcement: serialize_announcement_full(announcement),
      read: user_read.present?,
      read_at: user_read&.read_at
    }, status: :ok
  end

  # POST /api/v1/announcements/:id/read
  def mark_read
    announcement = Announcement.find_by(id: params[:id], deleted_at: nil)
    unless announcement
      return render json: { success: false, message: "Announcement not found" }, status: :not_found
    end

    read = announcement.announcement_reads.find_or_initialize_by(user_id: current_user.id)
    read.read_at ||= Time.current
    read.save!

    render json: {
      success: true,
      message: "Announcement marked as read",
      read_at: read.read_at,
      read_by_count: announcement.announcement_reads.count
    }, status: :ok
  end

  # POST /api/v1/announcements
  def create
    title = params[:title].to_s.strip
    content = params[:content].to_s.strip
    is_important = params[:is_important].to_s == "true" || params[:is_important] == true
    scheduled_at = params[:scheduled_at].present? ? Time.parse(params[:scheduled_at]) : nil

    unless title.present?
      return render json: {
        success: false,
        message: "Title is required"
      }, status: :unprocessable_entity
    end

    unless content.present?
      return render json: {
        success: false,
        message: "Content is required"
      }, status: :unprocessable_entity
    end

    announcement = Announcement.create!(
      title: title,
      content: content,
      is_pinned: is_important,
      created_by: current_user,
      published_at: scheduled_at.present? ? nil : Time.current
    )

    if scheduled_at.present? && scheduled_at > Time.current
      announcement.update!(scheduled_at: scheduled_at, published_at: nil)
    end

    announcement.reload
    announcement.announcement_reads.find_or_create_by!(user_id: current_user.id) do |read|
      read.read_at = Time.current
    end

    AuditLogService.log_announcement_created(announcement, created_by: current_user)

    NotificationService.notify_announcement(announcement, created_by: current_user)

    if announcement.published_at.present?
      render json: {
        success: true,
        message: "Announcement created and published",
        announcement: serialize_announcement_full(announcement)
      }, status: :created
    else
      render json: {
        success: true,
        message: "Announcement created (scheduled for later publication)",
        announcement: serialize_announcement_full(announcement)
      }, status: :created
    end
  end

  # PATCH /api/v1/announcements/:id
  def update
    announcement = Announcement.find_by(id: params[:id], deleted_at: nil)

    unless announcement
      return render json: {
        success: false,
        message: "Announcement not found"
      }, status: :not_found
    end

    changes = {}
    previous_title = announcement.title

    if params[:title].present? && announcement.title != params[:title].to_s.strip
      title = params[:title].to_s.strip
      unless title.present?
        return render json: {
          success: false,
          message: "Title cannot be empty"
        }, status: :unprocessable_entity
      end
      changes[:title] = { from: announcement.title, to: title }
      announcement.update!(title: title)
    end

    if params[:content].present? && announcement.content != params[:content].to_s.strip
      content = params[:content].to_s.strip
      unless content.present?
        return render json: {
          success: false,
          message: "Content cannot be empty"
        }, status: :unprocessable_entity
      end
      changes[:content] = { from: announcement.content.truncate(100), to: content.truncate(100) }
      announcement.update!(content: content)
    end

    if params[:is_important].present?
      new_value = params[:is_important].to_s == "true" || params[:is_important] == true
      if announcement.is_pinned != new_value
        changes[:is_important] = { from: announcement.is_pinned, to: new_value }
        announcement.update!(is_pinned: new_value)
      end
    end

    if changes.any?
      AuditLogService.log!(
        user: current_user,
        action: "announcement_updated",
        resource_type: "Announcement",
        resource_id: announcement.id,
        details: changes,
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Announcement updated",
        announcement: serialize_announcement_full(announcement)
      }, status: :ok
    else
      render json: {
        success: false,
        message: "No changes provided"
      }, status: :unprocessable_entity
    end
  end

  # POST /api/v1/announcements/:id/publish
  def publish
    announcement = Announcement.find_by(id: params[:id], deleted_at: nil)

    unless announcement
      return render json: {
        success: false,
        message: "Announcement not found"
      }, status: :not_found
    end

    unless announcement.published_at.nil?
      unless params[:publish_at].present?
        return render json: {
          success: false,
          message: "Announcement is already published"
        }, status: :conflict
      end

      publish_at = Time.parse(params[:publish_at])
      announcement.update!(
        published_at: publish_at,
        scheduled_at: nil
      )
    else
      announcement.update!(
        published_at: Time.current,
        scheduled_at: nil
      )
    end

    announcement.announcement_reads.find_or_create_by!(user_id: current_user.id) do |read|
      read.read_at = Time.current
    end

    AuditLogService.log_announcement_published(announcement, published_by: current_user)

    NotificationService.notify_announcement(announcement, created_by: current_user)

    render json: {
      success: true,
      message: "Announcement published",
      announcement: serialize_announcement_full(announcement)
    }, status: :ok
  end

  # POST /api/v1/announcements/:id/unpublish
  def unpublish
    announcement = Announcement.find_by(id: params[:id], deleted_at: nil)

    unless announcement
      return render json: {
        success: false,
        message: "Announcement not found"
      }, status: :not_found
    end

    unless announcement.published_at.present?
      return render json: {
        success: false,
        message: "Announcement is not published"
      }, status: :unprocessable_entity
    end

    announcement.update!(published_at: nil)

    AuditLogService.log!(
      user: current_user,
      action: "announcement_unpublished",
      resource_type: "Announcement",
      resource_id: announcement.id,
      details: { title: announcement.title },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Announcement unpublished",
      announcement: serialize_announcement_full(announcement)
    }, status: :ok
  end

  # POST /api/v1/announcements/:id/toggle-pin
  def toggle_pin
    announcement = Announcement.find_by(id: params[:id], deleted_at: nil)

    unless announcement
      return render json: {
        success: false,
        message: "Announcement not found"
      }, status: :not_found
    end

    new_pinned = !announcement.is_pinned
    announcement.update!(is_pinned: new_pinned)

    AuditLogService.log!(
      user: current_user,
      action: "announcement_#{new_pinned ? 'pinned' : 'unpinned'}",
      resource_type: "Announcement",
      resource_id: announcement.id,
      details: { title: announcement.title, is_pinned: new_pinned },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Announcement #{new_pinned ? 'pinned' : 'unpinned'}",
      announcement: serialize_announcement_full(announcement)
    }, status: :ok
  end

  # DELETE /api/v1/announcements/:id
  def destroy
    announcement = Announcement.find_by(id: params[:id], deleted_at: nil)

    unless announcement
      return render json: {
        success: false,
        message: "Announcement not found"
      }, status: :not_found
    end

    announcement.update!(deleted_at: Time.current)

    AuditLogService.log_announcement_deleted(announcement, deleted_by: current_user)

    render json: {
      success: true,
      message: "Announcement deleted"
    }, status: :ok
  end

  # GET /api/v1/announcements/:id/read-status
  def read_status
    announcement = Announcement.find_by(id: params[:id], deleted_at: nil)

    unless announcement
      return render json: {
        success: false,
        message: "Announcement not found"
      }, status: :not_found
    end

    read_user_ids = announcement.announcement_reads.pluck(:user_id)
    target_users = User.where(active: true).where(
      "role = 'admin' OR role = 'primary_admin' OR (role = 'member' AND EXISTS (
        SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = 'active'
      ))"
    )
    total_members = target_users.count
    unread_members = target_users.where.not(id: read_user_ids).count

    render json: {
      success: true,
      read_status: {
        announcement_id: announcement.id,
        total_members: total_members,
        unread_members: unread_members,
        read_percentage: total_members > 0 ? ((total_members - unread_members).to_f / total_members * 100).round(2) : 0
      }
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


  def serialize_announcement(announcement)
    {
      id: announcement.id,
      title: announcement.title,
      is_pinned: announcement.is_pinned,
      is_published: announcement.published_at.present?,
      published_at: announcement.published_at&.iso8601,
      scheduled_at: announcement.scheduled_at&.iso8601,
      created_by: {
        id: announcement.created_by.id,
        name: announcement.created_by.admin&.name || announcement.created_by.email
      },
      created_at: announcement.created_at,
      updated_at: announcement.updated_at
    }
  end

  def serialize_announcement_full(announcement)
    serialize_announcement(announcement).merge(
      content: announcement.content,
      read_by_count: announcement.announcement_reads.count
    )
  end
end
