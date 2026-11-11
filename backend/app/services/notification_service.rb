class NotificationService
  def self.notify_message(message:, sender:)
    all_user_ids = User
      .where.not(id: sender.id)
      .where(active: true)
      .where("role = 'admin' OR (role = 'member' AND EXISTS (
        SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = 'active'
      ))")
      .pluck(:id)

    all_user_ids.each do |user_id|
      Notification.create!(
        user_id: user_id,
        title: "New message",
        body: "#{sender.member&.name || sender.email} posted a new message",
        notification_type: "message",
        subject_type: "GroupMessage",
        subject_id: message.id,
        is_read: false
      )
    end

    send_email_notification_to_admins(
      subject: "New message in discussion",
      body: "#{sender.member&.name || sender.email} posted: #{message.content.truncate(200)}"
    )
  end

  def self.notify_reply(message, sender:, parent_message:)
    recipients = User
      .where(active: true)
      .where("role = 'admin' OR (role = 'member' AND EXISTS (
        SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = 'active'
      ))")
      .where.not(id: sender.id)
      .pluck(:id)

    recipients.each do |user_id|
      Notification.create!(
        user_id: user_id,
        title: "New reply",
        body: "#{sender.member&.name || sender.email} replied to a message",
        notification_type: "message_reply",
        subject_type: "GroupMessage",
        subject_id: message.id,
        is_read: false
      )
    end
  end

  def self.notify_forward(message, sender:, original_message:)
    recipients = User
      .where(active: true)
      .where("role = 'admin' OR (role = 'member' AND EXISTS (
        SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = 'active'
      ))")
      .where.not(id: sender.id)
      .pluck(:id)

    recipients.each do |user_id|
      Notification.create!(
        user_id: user_id,
        title: "Message forwarded",
        body: "#{sender.member&.name || sender.email} forwarded a message",
        notification_type: "message_forward",
        subject_type: "GroupMessage",
        subject_id: message.id,
        is_read: false
      )
    end
  end

  def self.notify_announcement(announcement, created_by:)
    return unless announcement.published_at.present?

    target_users = User
      .where(active: true)
      .where("role = 'admin' OR (role = 'member' AND EXISTS (
        SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = 'active'
      ))")
      .pluck(:id)

    target_users.each do |user_id|
      Notification.create!(
        user_id: user_id,
        title: "New announcement",
        body: announcement.title,
        notification_type: "announcement",
        subject_type: "Announcement",
        subject_id: announcement.id,
        is_read: false
      )
    end

    if announcement.published_at.present?
      send_announcement_email(announcement)
    end
  end

  def self.notify_event(event, created_by:)
    target_users = User
      .where(active: true)
      .where("role = 'admin' OR (role = 'member' AND EXISTS (
        SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = 'active'
      ))")
      .pluck(:id)

    target_users.each do |user_id|
      Notification.create!(
        user_id: user_id,
        title: "New event",
        body: event.title,
        notification_type: "event",
        subject_type: "Event",
        subject_id: event.id,
        is_read: false
      )
    end
  end

  def self.notify_task(task, created_by:)
    target_members = Member.active
      .where(academic_year: task.task_targets.pluck(:academic_year))
      .pluck(:user_id)

    target_members.each do |user_id|
      Notification.create!(
        user_id: user_id,
        title: "New task assigned",
        body: task.title,
        notification_type: "task",
        subject_type: "Task",
        subject_id: task.id,
        is_read: false
      )
    end
  end

  def self.notify_task_submission(task_submission, submitted_by:)
    task = task_submission.task
    admins = User.joins(:admin).where(users: { active: true })

    admins.each do |admin|
      Notification.create!(
        user_id: admin.id,
        title: "Task submission",
        body: "#{task_submission.member.name} submitted '#{task.title}'",
        notification_type: "submission_verification",
        subject_type: "TaskSubmissionStatus",
        subject_id: task_submission.id,
        is_read: false
      )
    end
  end

  def self.notify_task_verified(task_submission, verified_by:)
    member_user = task_submission.member.user

    Notification.create!(
      user_id: member_user.id,
      title: "Task verified",
      body: "Your submission for '#{task_submission.task.title}' has been verified",
      notification_type: "submission_verified",
      subject_type: "TaskSubmissionStatus",
      subject_id: task_submission.id,
      is_read: false
    )
  end

  def self.notify_borrow_request(borrow_record, requested_by:)
    admins = User.joins(:admin).where(users: { active: true })

    admins.each do |admin|
      Notification.create!(
        user_id: admin.id,
        title: "Borrow request",
        body: "#{requested_by.member.name} requested '#{borrow_record.inventory_item.name}'",
        notification_type: "inventory_approval",
        subject_type: "BorrowRecord",
        subject_id: borrow_record.id,
        is_read: false
      )
    end
  end

  def self.notify_borrow_rejected(borrow_record, rejected_by:, reason:)
    member_user = borrow_record.member.user

    Notification.create!(
      user_id: member_user.id,
      title: "Borrow request rejected",
      body: "Your request for '#{borrow_record.inventory_item.name}' was rejected#{reason.present? ? ": #{reason}" : "."}",
      notification_type: "inventory_rejected",
      subject_type: "BorrowRecord",
      subject_id: borrow_record.id,
      is_read: false
    )
  end

  def self.notify_borrow_approved(borrow_record, approved_by:)
    member_user = borrow_record.member.user

    Notification.create!(
      user_id: member_user.id,
      title: "Borrow approved",
      body: "Your request for '#{borrow_record.inventory_item.name}' has been approved",
      notification_type: "inventory_approved",
      subject_type: "BorrowRecord",
      subject_id: borrow_record.id,
      is_read: false
    )
  end

  def self.notify_return_confirmed(return_request, confirmed_by:)
    member_user = return_request.member.user
    borrow_record = return_request.borrow_record

    Notification.create!(
      user_id: member_user.id,
      title: "Return confirmed",
      body: "Your return of '#{borrow_record.inventory_item.name}' has been confirmed",
      notification_type: "return_confirmed",
      subject_type: "ReturnRequest",
      subject_id: return_request.id,
      is_read: false
    )
  end

  def self.notify_password_reset_request(user)
    Notification.create!(
      user_id: user.id,
      title: "Password reset requested",
      body: "A password reset was requested for your account",
      notification_type: "password_reset",
      subject_type: "User",
      subject_id: user.id,
      is_read: false
    )
  end

  def self.notify_password_changed(user)
    Notification.create!(
      user_id: user.id,
      title: "Password changed",
      body: "Your password has been changed successfully",
      notification_type: "password_changed",
      subject_type: "User",
      subject_id: user.id,
      is_read: false
    )
  end

  def self.notify_admin_created(admin, created_by:)
    notify_admin_of_action(
      recipients: User.joins(:admin).where(admins: { is_primary: true }, users: { active: true }),
      title: "New admin created",
      body: "#{created_by.admin.name} created admin account for #{admin.name} (#{admin.user.email})",
      type: "administrative_update",
      subject_type: "Admin",
      subject_id: admin.id
    )
  end

  def self.notify_admin_removed(admin, removed_by:)
    notify_admin_of_action(
      recipients: User.joins(:admin).where(admins: { is_primary: true }, users: { active: true }),
      title: "Admin removed",
      body: "#{removed_by.admin.name} removed admin account for #{admin.name} (#{admin.user.email})",
      type: "administrative_update",
      subject_type: "Admin",
      subject_id: admin.id
    )
  end

  def self.notify_member_status_change(member, changed_by:, new_status:)
    notify_admin_of_action(
      recipients: User.joins(:admin).where(users: { active: true }),
      title: "Member status changed",
      body: "#{changed_by.admin.name} changed #{member.name}'s status to #{new_status}",
      type: "administrative_update",
      subject_type: "Member",
      subject_id: member.id
    )
  end

  def self.notify_contact_message_received(message)
    admins = User.joins(:admin).where(users: { active: true })

    admins.each do |admin|
      Notification.create!(
        user_id: admin.id,
        title: "Contact message received",
        body: "New message from #{message.member.name}: #{message.subject}",
        notification_type: "contact_message",
        subject_type: "ContactMessage",
        subject_id: message.id,
        is_read: false
      )
    end
  end

  private

  def self.send_email_notification_to_admins(subject:, body:)
    admin_users = User.joins(:admin).where(users: { active: true })

    return unless defined?(AdminMailer)

    admin_users.each do |admin|
      safe_deliver { AdminMailer.notification(admin, subject, body).deliver_later }
    end
  end

  def self.send_announcement_email(announcement)
    target_users = User
      .where(active: true)
      .where("role = 'admin' OR (role = 'member' AND EXISTS (
        SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = 'active'
      ))")
      .pluck(:id)

    return unless defined?(UserMailer)

    target_users.each do |user_id|
      safe_deliver { UserMailer.announcement_notification(User.find(user_id), announcement).deliver_later }
    end
  end

  def self.safe_deliver
    yield
  rescue => e
    Rails.logger.warn("Email notification failed: #{e.class} - #{e.message}")
  end

  def self.notify_admin_of_action(recipients:, title:, body:, type:, subject_type:, subject_id:)
    recipients.each do |admin|
      Notification.create!(
        user_id: admin.id,
        title: title,
        body: body,
        notification_type: type,
        subject_type: subject_type,
        subject_id: subject_id,
        is_read: false
      )
    end
  end
end
