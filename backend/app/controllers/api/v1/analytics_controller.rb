class Api::V1::AnalyticsController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :require_admin!

  # GET /api/v1/analytics/dashboard
  def dashboard
    render json: {
      success: true,
      dashboard: build_dashboard_data
    }, status: :ok
  end

  # GET /api/v1/analytics/members
  def members
    total_members = Member.active.count
    active_members = Member.active.where(status: "active").count
    inactive_members = Member.where(status: "inactive").count
    deactivated_members = Member.where.not(deleted_at: nil).count

    academic_year_breakdown = Member.active
      .group(:academic_year)
      .select("academic_year, COUNT(*) AS count")
      .order(academic_year: :desc)

    branch_breakdown = Member.active
      .group(:branch)
      .select("branch, COUNT(*) AS count")
      .order("count DESC")

    section_breakdown = Member.active
      .group(:section)
      .select("section, COUNT(*) AS count")
      .order("count DESC")

    recent_registrations = Member.active
      .order(created_at: :desc)
      .limit(10)
      .select("id, name, login_id, email, branch, academic_year, created_at")

    render json: {
      success: true,
      members: {
        total: total_members,
        active: active_members,
        inactive: inactive_members,
        deactivated: deactivated_members,
        by_academic_year: academic_year_breakdown.map { |ay| { academic_year: ay.academic_year, count: ay.count } },
        by_branch: branch_breakdown.map { |b| { branch: b.branch, count: b.count } },
        by_section: section_breakdown.map { |s| { section: s.section, count: s.count } },
        recent_registrations: recent_registrations.map { |m|
          {
            id: m.id,
            name: m.name,
            login_id: m.login_id,
            email: m.email,
            branch: m.branch,
            academic_year: m.academic_year,
            created_at: m.created_at
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/analytics/attendance
  def attendance
    date_from = params[:date_from].present? ? Date.parse(params[:date_from]) : 30.days.ago.to_date
    date_to = params[:date_to].present? ? Date.parse(params[:date_to]) : Date.current

    total_login_events = AttendanceLog
      .where(entry_date: date_from..date_to)
      .where(exit_time: nil)
      .count

    total_logout_events = AttendanceLog
      .where(entry_date: date_from..date_to)
      .where.not(exit_time: nil)
      .count

    unique_members_present = AttendanceLog
      .where(entry_date: date_from..date_to)
      .distinct
      .count(:member_id)

    daily_data = AttendanceLog
      .where(entry_date: date_from..date_to)
      .group(:entry_date)
      .select(
        "entry_date",
        "COUNT(*) AS total_events",
        "COUNT(CASE WHEN exit_time IS NULL THEN 1 END) AS current_present",
        "COUNT(DISTINCT member_id) AS unique_members"
      )
      .order(entry_date: :desc)

    current_occupancy = AttendanceLog
      .where(entry_date: Date.current, exit_time: nil)
      .count

    branch_stats = AttendanceLog
      .joins(:member)
      .where(entry_date: date_from..date_to)
      .group("members.branch")
      .select("members.branch, COUNT(*) AS events, COUNT(DISTINCT attendance_logs.member_id) AS unique_members")
      .order("events DESC")

    render json: {
      success: true,
      attendance: {
        summary: {
          total_login_events: total_login_events,
          total_logout_events: total_logout_events,
          unique_members_present: unique_members_present,
          date_range: { from: date_from, to: date_to }
        },
        current_occupancy: {
          present_now: current_occupancy,
          date: Date.current,
          timestamp: Time.current
        },
        daily_summary: daily_data.map { |d|
          {
            date: d.entry_date.to_s,
            total_events: d.total_events,
            current_present: d.current_present,
            unique_members: d.unique_members
          }
        },
        by_branch: branch_stats.map { |b|
          {
            branch: b.branch,
            events: b.events,
            unique_members: b.unique_members
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/analytics/inventory
  def inventory_stats
    total_items = InventoryItem.where(is_active: true, deleted_at: nil).count
    total_stock = InventoryItem.where(is_active: true, deleted_at: nil).sum(:total_quantity)
    available_stock = InventoryItem.where(is_active: true, deleted_at: nil).sum(:available_quantity)
    borrowed_stock = total_stock - available_stock

    categories = InventoryItem
      .where(is_active: true, deleted_at: nil)
      .group(:category)
      .select("category, COUNT(*) AS items, SUM(total_quantity) AS total, SUM(available_quantity) AS available")
      .order("total DESC")

    low_stock_items = InventoryItem
      .where(is_active: true, deleted_at: nil)
      .where("available_quantity <= ?", 3)
      .order(available_quantity: :asc)
      .limit(10)

    recent_imports = InventoryImport
      .order(created_at: :desc)
      .limit(10)

    active_borrows = BorrowRecord
      .where(status: "approved")
      .includes(:member, :inventory_item)
      .order(due_date: :asc)

    overdue_items = active_borrows.where("due_date < ?", Date.current)

    render json: {
      success: true,
      inventory: {
        summary: {
          total_items: total_items,
          total_stock: total_stock,
          available_stock: available_stock,
          borrowed_stock: borrowed_stock
        },
        by_category: categories.map { |c|
          {
            category: c.category,
            items: c.items,
            total_quantity: c.total,
            available_quantity: c.available
          }
        },
        low_stock_items: low_stock_items.map { |i|
          {
            id: i.id,
            name: i.name,
            category: i.category,
            available_quantity: i.available_quantity,
            total_quantity: i.total_quantity
          }
        },
        active_borrows_count: active_borrows.count,
        overdue_borrows_count: overdue_items.count,
        recent_imports: recent_imports.map { |i|
          {
            id: i.id,
            filename: i.filename,
            total_rows: i.total_rows,
            imported_rows: i.imported_rows,
            error_rows: i.error_rows,
            status: i.status,
            created_at: i.created_at
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/analytics/tasks
  def task_stats
    total_tasks = Task.where(deleted_at: nil).count
    published_tasks = Task.where.not(published_at: nil, deleted_at: nil).count
    overdue_tasks = Task
      .where.not(published_at: nil, deleted_at: nil)
      .where("deadline < ?", Date.current)
      .count

    pending_submissions = TaskSubmissionStatus.where(status: "submitted").count
    verified_submissions = TaskSubmissionStatus.where(status: "verified").count
    rejected_submissions = TaskSubmissionStatus.where(status: "rejected").count

    tasks_by_type = Task
      .where(deleted_at: nil)
      .group(:task_type)
      .select("task_type, COUNT(*) AS count")
      .order("count DESC")

    tasks_by_year = Task
      .where(deleted_at: nil)
      .joins(:task_targets)
      .group("task_targets.academic_year")
      .select("task_targets.academic_year, COUNT(*) AS count")
      .order("count DESC")

    recent_tasks = Task
      .where(deleted_at: nil)
      .order(published_at: :desc)
      .limit(10)

    render json: {
      success: true,
      tasks: {
        summary: {
          total_tasks: total_tasks,
          published_tasks: published_tasks,
          overdue_tasks: overdue_tasks,
          pending_submissions: pending_submissions,
          verified_submissions: verified_submissions,
          rejected_submissions: rejected_submissions
        },
        by_type: tasks_by_type.map { |t| { task_type: t.task_type, count: t.count } },
        by_academic_year: tasks_by_year.map { |t| { academic_year: t.academic_year, count: t.count } },
        recent_tasks: recent_tasks.map { |t|
          {
            id: t.id,
            title: t.title,
            task_type: t.task_type,
            deadline: t.deadline.to_s,
            published_at: t.published_at,
            submission_count: TaskSubmissionStatus.where(task_id: t.id).count
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/analytics/announcements
  def announcement_stats
    total_announcements = Announcement.where(deleted_at: nil).count
    published_announcements = Announcement.where.not(published_at: nil, deleted_at: nil).count
    pinned_announcements = Announcement.where(is_pinned: true, deleted_at: nil).count

    read_analytics = Announcement
      .where(published_at: ..Date.current, deleted_at: nil)
      .left_joins(:announcement_reads)
      .group(:id)
      .select(
        "announcements.id",
        "announcements.title",
        "announcements.published_at",
        "COUNT(DISTINCT announcement_reads.user_id) AS read_count"
      )
      .order(published_at: :desc)

    total_reads = read_analytics.to_a.sum { |a| a.read_count.to_i }
    total_members = User.where(active: true).where(role: %w[admin primary_admin member]).count

    render json: {
      success: true,
      announcements: {
        summary: {
          total_announcements: total_announcements,
          published_announcements: published_announcements,
          pinned_announcements: pinned_announcements,
          total_reads: total_reads
        },
        recent: read_analytics.limit(10).map { |a|
          {
            id: a.id,
            title: a.title,
            published_at: a.published_at,
            read_count: a.read_count,
            read_percentage: total_members > 0 ? ((a.read_count.to_f / total_members * 100)).round(2) : 0
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/analytics/notifications
  def notification_stats
    total_notifications = Notification.where(user_id: User.active.select(:id)).count
    unread_notifications = Notification
      .where(user_id: User.active.select(:id), is_read: false)
      .count

    notifications_by_type = Notification
      .where(user_id: User.active.select(:id))
      .group(:notification_type)
      .select("notification_type, COUNT(*) AS count")
      .order("count DESC")

    recent_notifications = Notification
      .where(user_id: User.active.select(:id))
      .order(created_at: :desc)
      .limit(20)

    render json: {
      success: true,
      notifications: {
        total: total_notifications,
        unread: unread_notifications,
        by_type: notifications_by_type.map { |n| { notification_type: n.notification_type, count: n.count } },
        recent: recent_notifications.map { |n|
          {
            id: n.id,
            title: n.title,
            body: n.body,
            notification_type: n.notification_type,
            is_read: n.is_read,
            created_at: n.created_at
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/analytics/contact
  def contact_stats
    total_messages = ContactMessage.count
    new_messages = ContactMessage.where(status: "new").count
    resolved_messages = ContactMessage.where(status: "resolved").count
    closed_messages = ContactMessage.where(status: "closed").count

    messages_by_category = ContactMessage
      .group(:category)
      .select("category, COUNT(*) AS count")
      .order("count DESC")

    messages_by_status = ContactMessage
      .group(:status)
      .select("status, COUNT(*) AS count")
      .order("count DESC")

    recent_messages = ContactMessage
      .order(created_at: :desc)
      .limit(20)
      .includes(:member)

    render json: {
      success: true,
      contact: {
        summary: {
          total_messages: total_messages,
          new_messages: new_messages,
          in_progress: ContactMessage.where(status: "in_progress").count,
          resolved_messages: resolved_messages,
          closed_messages: closed_messages
        },
        by_category: messages_by_category.map { |c| { category: c.category, count: c.count } },
        by_status: messages_by_status.map { |s| { status: s.status, count: s.count } },
        recent: recent_messages.map { |m|
          {
            id: m.id,
            member_name: m.member.name,
            member_login_id: m.member.login_id,
            subject: m.subject,
            category: m.category,
            status: m.status,
            created_at: m.created_at
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/analytics/overview
  def overview
    render json: {
      success: true,
      overview: {
        members: {
          total: Member.active.count,
          active_today: Member.active.where(created_at: Date.current.beginning_of_day..Date.current.end_of_day).count
        },
        attendance: {
          present_now: AttendanceLog.where(entry_date: Date.current, exit_time: nil).count
        },
        inventory: {
          total_items: InventoryItem.where(is_active: true, deleted_at: nil).count,
          total_available: InventoryItem.where(is_active: true, deleted_at: nil).sum(:available_quantity)
        },
        tasks: {
          pending: TaskSubmissionStatus.where(status: "submitted").count,
          overdue: Task.where(deadline: ..Date.current, deleted_at: nil).count
        },
        announcements: {
          published: Announcement.where.not(published_at: nil, deleted_at: nil).count
        },
        notifications: {
          unread: Notification.where(user_id: User.active.select(:id), is_read: false).count
        },
        contact_messages: {
          new: ContactMessage.where(status: "new").count
        }
      }
    }, status: :ok
  end

  private

  def require_admin!
    return if current_user.admin?

    render json: {
      success: false,
      message: "Administrator access required"
    }, status: :forbidden
  end

  def total_members
    @total_members ||= Member.active.count
  end

  def build_dashboard_data
    {
      summary: {
        total_members: Member.active.count,
        active_today: Member.active.where(created_at: Date.current.beginning_of_day..Date.current.end_of_day).count,
        present_now: AttendanceLog.where(entry_date: Date.current, exit_time: nil).count,
        total_items: InventoryItem.where(is_active: true, deleted_at: nil).count,
        available_items: InventoryItem.where(is_active: true, deleted_at: nil).where("available_quantity > 0").count,
        pending_tasks: TaskSubmissionStatus.where(status: "submitted").count,
        pending_borrow_requests: BorrowRecord.where(status: "pending").count,
        pending_return_requests: ReturnRequest.where(status: "pending").count,
        new_contact_messages: ContactMessage.where(status: "new").count,
        unread_notifications: Notification.where(user_id: User.active.select(:id), is_read: false).count
      },
      trends: {
        members_by_academic_year: Member.active.group(:academic_year).count,
        members_by_branch: Member.active.group(:branch).count,
        inventory_by_category: InventoryItem.where(is_active: true, deleted_at: nil).group(:category).count
      }
    }
  end
end
