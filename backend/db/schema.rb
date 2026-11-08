# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# This file is the source Rails uses to define your schema when running `bin/rails
# db:schema:load`. When creating a new database, `bin/rails db:schema:load` tends to
# be faster and is potentially less error prone than running all of your
# migrations from scratch. Old migrations may fail to apply correctly if those
# migrations use external dependencies or application code.
#
# It's strongly recommended that you check this file into your version control system.

ActiveRecord::Schema[8.1].define(version: 2026_11_04_063207) do
  # These are extensions that must be enabled in order to support this database
  enable_extension "pg_catalog.plpgsql"
  enable_extension "pgcrypto"

  create_table "academic_years", force: :cascade do |t|
    t.integer "year", null: false
    t.string "label", null: false
    t.boolean "is_current", default: false, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["year"], name: "index_academic_years_on_year", unique: true
  end

  create_table "admin_transfers", force: :cascade do |t|
    t.bigint "from_admin_id", null: false
    t.bigint "to_admin_id", null: false
    t.string "status", default: "pending", null: false
    t.datetime "initiated_at", null: false
    t.datetime "completed_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["from_admin_id"], name: "index_admin_transfers_on_from_admin_id"
    t.index ["to_admin_id"], name: "index_admin_transfers_on_to_admin_id"
  end

  create_table "admins", force: :cascade do |t|
    t.bigint "user_id", null: false
    t.string "name", null: false
    t.boolean "is_primary", default: false, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["user_id"], name: "index_admins_on_user_id", unique: true
  end

  create_table "announcement_reads", force: :cascade do |t|
    t.bigint "announcement_id", null: false
    t.bigint "user_id", null: false
    t.datetime "read_at", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["announcement_id", "user_id"], name: "index_announcement_reads_on_announcement_id_and_user_id", unique: true
    t.index ["announcement_id"], name: "index_announcement_reads_on_announcement_id"
    t.index ["user_id"], name: "index_announcement_reads_on_user_id"
  end

  create_table "announcements", force: :cascade do |t|
    t.string "title", null: false
    t.text "content", null: false
    t.boolean "is_pinned", default: false, null: false
    t.datetime "published_at"
    t.datetime "scheduled_at"
    t.bigint "created_by_id", null: false
    t.datetime "deleted_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_by_id"], name: "index_announcements_on_created_by_id"
    t.index ["deleted_at"], name: "index_announcements_on_deleted_at"
    t.index ["published_at"], name: "index_announcements_on_published_at"
  end

  create_table "attendance_logs", force: :cascade do |t|
    t.bigint "member_id", null: false
    t.date "entry_date", null: false
    t.datetime "entry_time", null: false
    t.datetime "exit_time"
    t.string "source", default: "manual", null: false
    t.string "device_reference"
    t.bigint "created_by_id"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_by_id"], name: "index_attendance_logs_on_created_by_id"
    t.index ["entry_date"], name: "index_attendance_logs_on_entry_date"
    t.index ["member_id", "entry_date"], name: "index_attendance_logs_on_member_id_and_entry_date"
    t.index ["member_id"], name: "index_attendance_logs_on_member_id"
  end

  create_table "audit_logs", force: :cascade do |t|
    t.bigint "user_id"
    t.string "action", null: false
    t.string "resource_type", null: false
    t.bigint "resource_id"
    t.jsonb "details", default: {}
    t.string "ip_address"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_at"], name: "index_audit_logs_on_created_at"
    t.index ["resource_type", "resource_id"], name: "index_audit_logs_on_resource_type_and_resource_id"
    t.index ["user_id"], name: "index_audit_logs_on_user_id"
  end

  create_table "biometric_enrollments", force: :cascade do |t|
    t.bigint "member_id", null: false
    t.bigint "device_registration_id", null: false
    t.string "device_user_id", null: false
    t.string "status", default: "pending", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["device_registration_id"], name: "index_biometric_enrollments_on_device_registration_id"
    t.index ["member_id", "device_registration_id"], name: "idx_biometric_enrollment_unique", unique: true
    t.index ["member_id"], name: "index_biometric_enrollments_on_member_id"
  end

  create_table "borrow_records", force: :cascade do |t|
    t.bigint "member_id", null: false
    t.bigint "inventory_item_id", null: false
    t.string "status", default: "borrowed", null: false
    t.datetime "borrowed_at", null: false
    t.date "due_date", null: false
    t.datetime "returned_at"
    t.string "condition_on_return"
    t.string "idempotency_key"
    t.bigint "approved_by_id"
    t.string "approval_reason"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["approved_by_id"], name: "index_borrow_records_on_approved_by_id"
    t.index ["idempotency_key"], name: "index_borrow_records_on_idempotency_key", unique: true, where: "(idempotency_key IS NOT NULL)"
    t.index ["inventory_item_id"], name: "index_borrow_records_on_inventory_item_id"
    t.index ["member_id", "status"], name: "index_borrow_records_on_member_id_and_status"
    t.index ["member_id"], name: "index_borrow_records_on_member_id"
    t.index ["status", "due_date"], name: "index_borrow_records_on_status_and_due_date"
  end

  create_table "contact_messages", force: :cascade do |t|
    t.bigint "member_id", null: false
    t.string "subject", null: false
    t.string "category", null: false
    t.text "message", null: false
    t.string "status", default: "new", null: false
    t.string "email_delivery_status"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["member_id"], name: "index_contact_messages_on_member_id"
    t.index ["status"], name: "index_contact_messages_on_status"
  end

  create_table "content_blocks", force: :cascade do |t|
    t.string "key", null: false
    t.string "title"
    t.text "content"
    t.integer "position", default: 0, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["key"], name: "index_content_blocks_on_key", unique: true
  end

  create_table "device_events", force: :cascade do |t|
    t.bigint "device_registration_id", null: false
    t.string "event_type", null: false
    t.string "idempotency_key", null: false
    t.string "member_identifier"
    t.string "hmac_signature"
    t.datetime "event_timestamp", null: false
    t.jsonb "payload", default: {}
    t.string "processing_status", default: "pending", null: false
    t.text "processing_error"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["device_registration_id"], name: "index_device_events_on_device_registration_id"
    t.index ["idempotency_key"], name: "index_device_events_on_idempotency_key", unique: true
    t.index ["processing_status"], name: "index_device_events_on_processing_status"
  end

  create_table "device_registrations", force: :cascade do |t|
    t.string "name", null: false
    t.string "device_type", null: false
    t.string "api_key_digest", null: false
    t.string "status", default: "active", null: false
    t.datetime "last_seen_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["api_key_digest"], name: "index_device_registrations_on_api_key_digest", unique: true
  end

  create_table "events", force: :cascade do |t|
    t.string "title", null: false
    t.text "description"
    t.string "event_type", null: false
    t.date "date", null: false
    t.time "start_time", null: false
    t.time "end_time", null: false
    t.string "venue"
    t.text "external_links", default: [], array: true
    t.bigint "created_by_id", null: false
    t.datetime "deleted_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_by_id"], name: "index_events_on_created_by_id"
    t.index ["date"], name: "index_events_on_date"
    t.index ["deleted_at"], name: "index_events_on_deleted_at"
  end

  create_table "faq_categories", force: :cascade do |t|
    t.string "name", null: false
    t.string "slug", null: false
    t.integer "position", default: 0, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["slug"], name: "index_faq_categories_on_slug", unique: true
  end

  create_table "faqs", force: :cascade do |t|
    t.bigint "faq_category_id", null: false
    t.string "question", null: false
    t.text "answer", null: false
    t.integer "position", default: 0, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["faq_category_id"], name: "index_faqs_on_faq_category_id"
  end

  create_table "group_messages", force: :cascade do |t|
    t.bigint "user_id", null: false
    t.text "content"
    t.bigint "reply_to_id"
    t.bigint "forwarded_from_id"
    t.boolean "is_edited", default: false, null: false
    t.boolean "is_deleted", default: false, null: false
    t.boolean "is_pinned", default: false, null: false
    t.string "message_type", default: "text", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_at"], name: "index_group_messages_on_created_at"
    t.index ["forwarded_from_id"], name: "index_group_messages_on_forwarded_from_id"
    t.index ["reply_to_id"], name: "index_group_messages_on_reply_to_id"
    t.index ["user_id"], name: "index_group_messages_on_user_id"
  end

  create_table "group_settings", force: :cascade do |t|
    t.string "permission", default: "everyone", null: false
    t.string "description"
    t.boolean "media_sharing_enabled", default: true, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
  end

  create_table "inventory_imports", force: :cascade do |t|
    t.string "filename", null: false
    t.integer "total_rows", null: false
    t.integer "imported_rows", default: 0, null: false
    t.integer "error_rows", default: 0, null: false
    t.string "status", default: "completed", null: false
    t.bigint "imported_by_id", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["imported_by_id"], name: "index_inventory_imports_on_imported_by_id"
  end

  create_table "inventory_items", force: :cascade do |t|
    t.string "name", null: false
    t.string "category", null: false
    t.text "description"
    t.integer "total_quantity", default: 0, null: false
    t.integer "available_quantity", default: 0, null: false
    t.string "storage_location"
    t.string "condition"
    t.boolean "is_active", default: true, null: false
    t.datetime "deleted_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["deleted_at"], name: "index_inventory_items_on_deleted_at"
    t.index ["is_active"], name: "index_inventory_items_on_is_active"
    t.index ["name", "category"], name: "index_inventory_items_on_name_and_category"
    t.check_constraint "available_quantity >= 0 AND available_quantity <= total_quantity", name: "check_available_quantity"
  end

  create_table "lab_rules_acceptances", force: :cascade do |t|
    t.bigint "member_id", null: false
    t.bigint "lab_rules_version_id", null: false
    t.datetime "accepted_at", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["lab_rules_version_id"], name: "index_lab_rules_acceptances_on_lab_rules_version_id"
    t.index ["member_id", "lab_rules_version_id"], name: "idx_rules_acceptance_unique", unique: true
    t.index ["member_id"], name: "index_lab_rules_acceptances_on_member_id"
  end

  create_table "lab_rules_versions", force: :cascade do |t|
    t.integer "version_number", null: false
    t.text "content", null: false
    t.boolean "is_current", default: false, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["version_number"], name: "index_lab_rules_versions_on_version_number", unique: true
  end

  create_table "link_previews", force: :cascade do |t|
    t.bigint "group_message_id", null: false
    t.string "url", null: false
    t.string "title"
    t.text "description"
    t.string "image_url"
    t.string "site_name"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["group_message_id"], name: "index_link_previews_on_group_message_id"
  end

  create_table "members", force: :cascade do |t|
    t.bigint "user_id", null: false
    t.string "name", null: false
    t.string "usn"
    t.string "roll_number"
    t.string "login_id", null: false
    t.integer "academic_year", null: false
    t.string "branch", null: false
    t.string "section"
    t.string "email", null: false
    t.string "phone", null: false
    t.boolean "must_change_password", default: true, null: false
    t.boolean "login_id_changed", default: false, null: false
    t.datetime "login_id_changed_at"
    t.string "status", default: "active", null: false
    t.string "github_url"
    t.string "linkedin_url"
    t.string "portfolio_url"
    t.text "bio"
    t.text "interests", default: [], array: true
    t.datetime "deleted_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.boolean "phone_verified", default: false, null: false
    t.index ["academic_year"], name: "index_members_on_academic_year"
    t.index ["branch"], name: "index_members_on_branch"
    t.index ["deleted_at"], name: "index_members_on_deleted_at"
    t.index ["login_id"], name: "index_members_on_login_id", unique: true
    t.index ["roll_number"], name: "index_members_on_roll_number", unique: true, where: "(roll_number IS NOT NULL)"
    t.index ["status"], name: "index_members_on_status"
    t.index ["user_id"], name: "index_members_on_user_id", unique: true
    t.index ["usn"], name: "index_members_on_usn", unique: true, where: "(usn IS NOT NULL)"
  end

  create_table "message_attachments", force: :cascade do |t|
    t.bigint "group_message_id", null: false
    t.string "filename", null: false
    t.string "content_type", null: false
    t.integer "byte_size", null: false
    t.string "storage_key", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["group_message_id"], name: "index_message_attachments_on_group_message_id"
  end

  create_table "message_pins", force: :cascade do |t|
    t.bigint "group_message_id", null: false
    t.bigint "pinned_by_id", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["group_message_id"], name: "index_message_pins_on_group_message_id"
    t.index ["pinned_by_id"], name: "index_message_pins_on_pinned_by_id"
  end

  create_table "message_reactions", force: :cascade do |t|
    t.bigint "group_message_id", null: false
    t.bigint "user_id", null: false
    t.string "emoji", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["group_message_id", "user_id", "emoji"], name: "idx_reaction_unique", unique: true
    t.index ["group_message_id"], name: "index_message_reactions_on_group_message_id"
    t.index ["user_id"], name: "index_message_reactions_on_user_id"
  end

  create_table "message_receipts", force: :cascade do |t|
    t.bigint "group_message_id", null: false
    t.bigint "user_id", null: false
    t.datetime "delivered_at"
    t.datetime "read_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["group_message_id", "user_id"], name: "idx_receipt_unique", unique: true
    t.index ["group_message_id"], name: "index_message_receipts_on_group_message_id"
    t.index ["user_id", "read_at"], name: "index_message_receipts_on_user_id_and_read_at"
    t.index ["user_id"], name: "index_message_receipts_on_user_id"
  end

  create_table "message_stars", force: :cascade do |t|
    t.bigint "group_message_id", null: false
    t.bigint "user_id", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["group_message_id", "user_id"], name: "index_message_stars_on_group_message_id_and_user_id", unique: true
    t.index ["group_message_id"], name: "index_message_stars_on_group_message_id"
    t.index ["user_id"], name: "index_message_stars_on_user_id"
  end

  create_table "notifications", force: :cascade do |t|
    t.bigint "user_id", null: false
    t.string "title", null: false
    t.text "body"
    t.string "notification_type", null: false
    t.string "subject_type"
    t.bigint "subject_id"
    t.boolean "is_read", default: false, null: false
    t.datetime "read_at"
    t.string "email_status"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["user_id", "is_read", "created_at"], name: "index_notifications_on_user_id_and_is_read_and_created_at"
    t.index ["user_id", "notification_type", "subject_id"], name: "idx_notification_dedup"
    t.index ["user_id"], name: "index_notifications_on_user_id"
  end

  create_table "otp_codes", force: :cascade do |t|
    t.bigint "user_id", null: false
    t.string "otp_digest", null: false
    t.string "delivery_method", null: false
    t.string "purpose", null: false
    t.integer "attempts", default: 0, null: false
    t.boolean "used", default: false, null: false
    t.datetime "expires_at", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["user_id", "purpose", "created_at"], name: "index_otp_codes_on_user_id_and_purpose_and_created_at"
    t.index ["user_id"], name: "index_otp_codes_on_user_id"
  end

  create_table "platform_settings", force: :cascade do |t|
    t.string "key", null: false
    t.text "value"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["key"], name: "index_platform_settings_on_key", unique: true
  end

  create_table "poll_options", force: :cascade do |t|
    t.bigint "poll_id", null: false
    t.string "text", null: false
    t.integer "position", default: 0, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["poll_id"], name: "index_poll_options_on_poll_id"
  end

  create_table "poll_votes", force: :cascade do |t|
    t.bigint "poll_id", null: false
    t.bigint "poll_option_id", null: false
    t.bigint "user_id", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["poll_id", "user_id", "poll_option_id"], name: "idx_poll_vote_unique", unique: true
    t.index ["poll_id"], name: "index_poll_votes_on_poll_id"
    t.index ["poll_option_id"], name: "index_poll_votes_on_poll_option_id"
    t.index ["user_id"], name: "index_poll_votes_on_user_id"
  end

  create_table "polls", force: :cascade do |t|
    t.bigint "group_message_id", null: false
    t.string "question", null: false
    t.boolean "is_multiple_choice", default: false, null: false
    t.datetime "closes_at"
    t.boolean "is_closed", default: false, null: false
    t.bigint "created_by_id", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_by_id"], name: "index_polls_on_created_by_id"
    t.index ["group_message_id"], name: "index_polls_on_group_message_id"
  end

  create_table "return_requests", force: :cascade do |t|
    t.bigint "borrow_record_id", null: false
    t.bigint "member_id", null: false
    t.string "status", default: "pending", null: false
    t.datetime "confirmed_at"
    t.bigint "confirmed_by_id"
    t.string "condition_note"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["borrow_record_id"], name: "index_return_requests_on_borrow_record_id"
    t.index ["confirmed_by_id"], name: "index_return_requests_on_confirmed_by_id"
    t.index ["member_id"], name: "index_return_requests_on_member_id"
  end

  create_table "semesters", force: :cascade do |t|
    t.string "name", null: false
    t.date "start_date", null: false
    t.date "end_date", null: false
    t.boolean "is_current", default: false, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
  end

  create_table "sessions", force: :cascade do |t|
    t.bigint "user_id", null: false
    t.string "token_digest", null: false
    t.string "ip_address"
    t.string "user_agent"
    t.datetime "last_active_at", null: false
    t.datetime "expires_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["token_digest"], name: "index_sessions_on_token_digest", unique: true
    t.index ["user_id", "last_active_at"], name: "index_sessions_on_user_id_and_last_active_at"
    t.index ["user_id"], name: "index_sessions_on_user_id"
  end

  create_table "task_submission_statuses", force: :cascade do |t|
    t.bigint "task_id", null: false
    t.bigint "member_id", null: false
    t.string "status", default: "pending", null: false
    t.datetime "submitted_at"
    t.datetime "verified_at"
    t.bigint "verified_by_id"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["member_id"], name: "index_task_submission_statuses_on_member_id"
    t.index ["task_id", "member_id"], name: "idx_task_submission_unique", unique: true
    t.index ["task_id"], name: "index_task_submission_statuses_on_task_id"
    t.index ["verified_by_id"], name: "index_task_submission_statuses_on_verified_by_id"
  end

  create_table "task_targets", force: :cascade do |t|
    t.bigint "task_id", null: false
    t.integer "academic_year", null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["task_id", "academic_year"], name: "index_task_targets_on_task_id_and_academic_year", unique: true
    t.index ["task_id"], name: "index_task_targets_on_task_id"
  end

  create_table "tasks", force: :cascade do |t|
    t.string "title", null: false
    t.text "description"
    t.string "task_type", null: false
    t.datetime "published_at"
    t.datetime "deadline", null: false
    t.text "instructions"
    t.string "external_link"
    t.string "required_attachments"
    t.bigint "created_by_id", null: false
    t.datetime "deleted_at"
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["created_by_id"], name: "index_tasks_on_created_by_id"
    t.index ["deadline"], name: "index_tasks_on_deadline"
    t.index ["deleted_at"], name: "index_tasks_on_deleted_at"
  end

  create_table "users", force: :cascade do |t|
    t.string "email", null: false
    t.string "phone"
    t.string "password_digest"
    t.string "role", default: "member", null: false
    t.boolean "active", default: true, null: false
    t.datetime "created_at", null: false
    t.datetime "updated_at", null: false
    t.index ["email"], name: "index_users_on_email", unique: true
    t.index ["phone"], name: "index_users_on_phone", unique: true, where: "(phone IS NOT NULL)"
    t.index ["role"], name: "index_users_on_role"
  end

  add_foreign_key "admin_transfers", "admins", column: "from_admin_id"
  add_foreign_key "admin_transfers", "admins", column: "to_admin_id"
  add_foreign_key "admins", "users"
  add_foreign_key "announcement_reads", "announcements"
  add_foreign_key "announcement_reads", "users"
  add_foreign_key "announcements", "users", column: "created_by_id"
  add_foreign_key "attendance_logs", "members"
  add_foreign_key "attendance_logs", "users", column: "created_by_id"
  add_foreign_key "audit_logs", "users"
  add_foreign_key "biometric_enrollments", "device_registrations"
  add_foreign_key "biometric_enrollments", "members"
  add_foreign_key "borrow_records", "inventory_items"
  add_foreign_key "borrow_records", "members"
  add_foreign_key "borrow_records", "users", column: "approved_by_id"
  add_foreign_key "contact_messages", "members"
  add_foreign_key "device_events", "device_registrations"
  add_foreign_key "events", "users", column: "created_by_id"
  add_foreign_key "faqs", "faq_categories"
  add_foreign_key "group_messages", "group_messages", column: "forwarded_from_id"
  add_foreign_key "group_messages", "group_messages", column: "reply_to_id"
  add_foreign_key "group_messages", "users"
  add_foreign_key "inventory_imports", "users", column: "imported_by_id"
  add_foreign_key "lab_rules_acceptances", "lab_rules_versions"
  add_foreign_key "lab_rules_acceptances", "members"
  add_foreign_key "link_previews", "group_messages"
  add_foreign_key "members", "users"
  add_foreign_key "message_attachments", "group_messages"
  add_foreign_key "message_pins", "group_messages"
  add_foreign_key "message_pins", "users", column: "pinned_by_id"
  add_foreign_key "message_reactions", "group_messages"
  add_foreign_key "message_reactions", "users"
  add_foreign_key "message_receipts", "group_messages"
  add_foreign_key "message_receipts", "users"
  add_foreign_key "message_stars", "group_messages"
  add_foreign_key "message_stars", "users"
  add_foreign_key "notifications", "users"
  add_foreign_key "otp_codes", "users"
  add_foreign_key "poll_options", "polls"
  add_foreign_key "poll_votes", "poll_options"
  add_foreign_key "poll_votes", "polls"
  add_foreign_key "poll_votes", "users"
  add_foreign_key "polls", "group_messages"
  add_foreign_key "polls", "users", column: "created_by_id"
  add_foreign_key "return_requests", "borrow_records"
  add_foreign_key "return_requests", "members"
  add_foreign_key "return_requests", "users", column: "confirmed_by_id"
  add_foreign_key "sessions", "users"
  add_foreign_key "task_submission_statuses", "members"
  add_foreign_key "task_submission_statuses", "tasks"
  add_foreign_key "task_submission_statuses", "users", column: "verified_by_id"
  add_foreign_key "task_targets", "tasks"
  add_foreign_key "tasks", "users", column: "created_by_id"
end
