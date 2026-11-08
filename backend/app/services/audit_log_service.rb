class AuditLogService
  def self.log!(user:, action:, resource_type:, resource_id: nil, details: {}, ip_address: nil)
    AuditLog.create!(
      user: user,
      action: action.to_s,
      resource_type: resource_type.to_s,
      resource_id: resource_id,
      details: details.with_indifferent_access,
      ip_address: ip_address
    )
  rescue => e
    Rails.logger.error("Audit log failed: #{e.class} - #{e.message}")
  end

  def self.log_registration(member, ip_address: nil)
    log!(
      user: nil,
      action: "member_registered",
      resource_type: "Member",
      resource_id: member.id,
      details: {
        login_id: member.login_id,
        name: member.name,
        email: member.email,
        academic_year: member.academic_year,
        branch: member.branch,
        section: member.section
      },
      ip_address: ip_address
    )
  end

  def self.log_login(user, ip_address:, login_type: nil)
    log!(
      user: user,
      action: "login",
      resource_type: "User",
      resource_id: user.id,
      details: { login_type: login_type },
      ip_address: ip_address
    )
  end

  def self.log_logout(user, ip_address:)
    log!(
      user: user,
      action: "logout",
      resource_type: "User",
      resource_id: user.id,
      ip_address: ip_address
    )
  end

  def self.log_password_change(user, ip_address:)
    log!(
      user: user,
      action: "password_changed",
      resource_type: "User",
      resource_id: user.id,
      ip_address: ip_address
    )
  end

  def self.log_admin_created(admin, created_by:, ip_address:)
    log!(
      user: created_by,
      action: "admin_created",
      resource_type: "Admin",
      resource_id: admin.id,
      details: {
        admin_name: admin.name,
        admin_email: admin.user.email,
        is_primary: admin.is_primary
      },
      ip_address: ip_address
    )
  end

  def self.log_admin_updated(admin, updated_by:, changes:, ip_address:)
    log!(
      user: updated_by,
      action: "admin_updated",
      resource_type: "Admin",
      resource_id: admin.id,
      details: {
        admin_name: admin.name,
        changes: changes
      },
      ip_address: ip_address
    )
  end

  def self.log_admin_removed(admin, removed_by:, ip_address:)
    log!(
      user: removed_by,
      action: "admin_removed",
      resource_type: "Admin",
      resource_id: admin.id,
      details: {
        admin_name: admin.name,
        admin_email: admin.user.email
      },
      ip_address: ip_address
    )
  end

  def self.log_member_status_change(member, changed_by:, new_status:, ip_address:)
    log!(
      user: changed_by,
      action: "member_status_changed",
      resource_type: "Member",
      resource_id: member.id,
      details: {
        member_login_id: member.login_id,
        member_name: member.name,
        previous_status: member.status,
        new_status: new_status
      },
      ip_address: ip_address
    )
  end

  def self.log_profile_update(member, updated_by:, changes:, ip_address:)
    log!(
      user: updated_by,
      action: "profile_updated",
      resource_type: "Member",
      resource_id: member.id,
      details: changes,
      ip_address: ip_address
    )
  end

  def self.log_biometric_enrollment(member, device_registration, enrolled_by:, ip_address:)
    log!(
      user: enrolled_by,
      action: "biometric_enrolled",
      resource_type: "BiometricEnrollment",
      resource_id: nil,
      details: {
        member_login_id: member.login_id,
        member_name: member.name,
        device_name: device_registration.name,
        device_user_id: nil
      },
      ip_address: ip_address
    )
  end

  def self.log_biometric_enrollment_confirmed(enrollment, confirmed_by:, device_user_id:, ip_address:)
    log!(
      user: confirmed_by,
      action: "biometric_enrollment_confirmed",
      resource_type: "BiometricEnrollment",
      resource_id: enrollment.id,
      details: {
        member_login_id: enrollment.member.login_id,
        member_name: enrollment.member.name,
        device_user_id: device_user_id
      },
      ip_address: ip_address
    )
  end

  def self.log_biometric_enrollment_removed(enrollment, removed_by:, ip_address:)
    log!(
      user: removed_by,
      action: "biometric_enrollment_removed",
      resource_type: "BiometricEnrollment",
      resource_id: enrollment.id,
      details: {
        member_login_id: enrollment.member.login_id,
        member_name: enrollment.member.name,
        device_user_id: enrollment.device_user_id
      },
      ip_address: ip_address
    )
  end

  def self.log_attendance_recorded(attendance_log, source:, ip_address:)
    log!(
      user: attendance_log.created_by,
      action: "attendance_recorded",
      resource_type: "AttendanceLog",
      resource_id: attendance_log.id,
      details: {
        member_login_id: attendance_log.member.login_id,
        member_name: attendance_log.member.name,
        entry_date: attendance_log.entry_date.to_s,
        entry_time: attendance_log.entry_time.to_s,
        source: source
      },
      ip_address: ip_address
    )
  end

  def self.log_attendance_manual_override(attendance_log, overridden_by:, reason:, ip_address:)
    log!(
      user: overridden_by,
      action: "attendance_override",
      resource_type: "AttendanceLog",
      resource_id: attendance_log.id,
      details: {
        member_login_id: attendance_log.member.login_id,
        previous_entry_time: attendance_log.entry_time.to_s,
        new_entry_time: attendance_log.entry_time.to_s,
        reason: reason
      },
      ip_address: ip_address
    )
  end

  def self.log_inventory_import(import, imported_by:)
    log!(
      user: imported_by,
      action: "inventory_imported",
      resource_type: "InventoryImport",
      resource_id: import.id,
      details: {
        filename: import.filename,
        total_rows: import.total_rows,
        imported_rows: import.imported_rows,
        error_rows: import.error_rows
      },
      ip_address: nil
    )
  end

  def self.log_borrow_request(borrow_record, requested_by:)
    log!(
      user: requested_by,
      action: "borrow_requested",
      resource_type: "BorrowRecord",
      resource_id: borrow_record.id,
      details: {
        member_login_id: borrow_record.member.login_id,
        item_name: borrow_record.inventory_item.name,
        item_category: borrow_record.inventory_item.category,
        quantity: borrow_record.quantity,
        due_date: borrow_record.due_date.to_s
      },
      ip_address: nil
    )
  end

  def self.log_borrow_approved(borrow_record, approved_by:, ip_address:)
    log!(
      user: approved_by,
      action: "borrow_approved",
      resource_type: "BorrowRecord",
      resource_id: borrow_record.id,
      details: {
        member_login_id: borrow_record.member.login_id,
        item_name: borrow_record.inventory_item.name,
        approved_reason: borrow_record.approval_reason
      },
      ip_address: ip_address
    )
  end

  def self.log_borrow_rejected(borrow_record, rejected_by:, reason:, ip_address:)
    log!(
      user: rejected_by,
      action: "borrow_rejected",
      resource_type: "BorrowRecord",
      resource_id: borrow_record.id,
      details: {
        member_login_id: borrow_record.member.login_id,
        item_name: borrow_record.inventory_item.name,
        rejection_reason: reason
      },
      ip_address: ip_address
    )
  end

  def self.log_return_confirmed(return_request, confirmed_by:, ip_address:)
    log!(
      user: confirmed_by,
      action: "return_confirmed",
      resource_type: "ReturnRequest",
      resource_id: return_request.id,
      details: {
        member_login_id: return_request.member.login_id,
        item_name: return_request.borrow_record.inventory_item.name,
        condition_note: return_request.condition_note
      },
      ip_address: ip_address
    )
  end

  def self.log_contact_email_change(previous_email:, new_email:, changed_by:, ip_address:)
    log!(
      user: changed_by,
      action: "contact_email_changed",
      resource_type: "PlatformSetting",
      resource_id: nil,
      details: {
        previous_email: previous_email,
        new_email: new_email
      },
      ip_address: ip_address
    )
  end

  def self.log_announcement_created(announcement, created_by:)
    log!(
      user: created_by,
      action: "announcement_created",
      resource_type: "Announcement",
      resource_id: announcement.id,
      details: {
        title: announcement.title
      },
      ip_address: nil
    )
  end

  def self.log_announcement_published(announcement, published_by:)
    log!(
      user: published_by,
      action: "announcement_published",
      resource_type: "Announcement",
      resource_id: announcement.id,
      details: {
        title: announcement.title,
        published_at: announcement.published_at.to_s
      },
      ip_address: nil
    )
  end

  def self.log_announcement_deleted(announcement, deleted_by:)
    log!(
      user: deleted_by,
      action: "announcement_deleted",
      resource_type: "Announcement",
      resource_id: announcement.id,
      details: {
        title: announcement.title
      },
      ip_address: nil
    )
  end

  def self.log_event_created(event, created_by:)
    log!(
      user: created_by,
      action: "event_created",
      resource_type: "Event",
      resource_id: event.id,
      details: {
        title: event.title,
        event_type: event.event_type,
        date: event.date.to_s
      },
      ip_address: nil
    )
  end

  def self.log_task_created(task, created_by:)
    log!(
      user: created_by,
      action: "task_created",
      resource_type: "Task",
      resource_id: task.id,
      details: {
        title: task.title,
        task_type: task.task_type,
        deadline: task.deadline.to_s
      },
      ip_address: nil
    )
  end

  def self.log_task_submission(task_submission, submitted_by:)
    log!(
      user: submitted_by,
      action: "task_submitted",
      resource_type: "TaskSubmissionStatus",
      resource_id: task_submission.id,
      details: {
        task_title: task_submission.task.title,
        member_login_id: task_submission.member.login_id
      },
      ip_address: nil
    )
  end

  def self.log_task_verified(task_submission, verified_by:)
    log!(
      user: verified_by,
      action: "task_verified",
      resource_type: "TaskSubmissionStatus",
      resource_id: task_submission.id,
      details: {
        task_title: task_submission.task.title,
        member_login_id: task_submission.member.login_id
      },
      ip_address: nil
    )
  end

  def self.log_message_sent(message, sent_by:)
    log!(
      user: sent_by,
      action: "message_sent",
      resource_type: "GroupMessage",
      resource_id: message.id,
      details: {
        reply_to_id: message.reply_to_id,
        forwarded_from_id: message.forwarded_from_id,
        message_type: message.message_type
      },
      ip_address: nil
    )
  end

  def self.log_message_edited(message, edited_by:)
    log!(
      user: edited_by,
      action: "message_edited",
      resource_type: "GroupMessage",
      resource_id: message.id,
      details: {
        original_content_length: message.content.length
      },
      ip_address: nil
    )
  end

  def self.log_message_deleted(message, deleted_by:)
    log!(
      user: deleted_by,
      action: "message_deleted",
      resource_type: "GroupMessage",
      resource_id: message.id,
      details: {
        original_content: message.content
      },
      ip_address: nil
    )
  end

  def self.log_membership_invite(invited_by:, member_data:, ip_address:)
    log!(
      user: invited_by,
      action: "membership_invited",
      resource_type: "Member",
      resource_id: nil,
      details: {
        email: member_data[:email],
        name: member_data[:name],
        academic_year: member_data[:academic_year],
        branch: member_data[:branch]
      },
      ip_address: ip_address
    )
  end
end
