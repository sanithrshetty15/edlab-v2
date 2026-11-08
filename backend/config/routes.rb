Rails.application.routes.draw do
  get "up" => "rails/health#show", as: :rails_health_check

  namespace :api do
    namespace :v1 do
      get "health", to: "health#show"

      # Authentication
      post "auth/member-login", to: "auth#member_login"
      post "auth/admin-login", to: "auth#admin_login"
      delete "auth/logout", to: "auth#logout"
      get "auth/me", to: "auth#me"
      patch "auth/change-password", to: "auth#change_password"
      post "auth/password-reset/request", to: "password_resets#create"
      post "auth/password-reset/complete", to: "password_resets#update"

      # Member registration and profile
      post "members/register", to: "members#register"
      get "members/me", to: "members#show_current"
      patch "members/me/profile", to: "members#update_profile"
      patch "members/me/accept-rules", to: "members#accept_lab_rules"
      post "members/me/phone-verification/request", to: "members#request_phone_verification"
      patch "members/me/phone-verification", to: "members#verify_phone"

      # Admin management (primary admin only)
      get "admins", to: "admins#index"
      get "admins/:id", to: "admins#show", constraints: { id: /[0-9]+/ }
      post "admins", to: "admins#create"
      patch "admins/:id", to: "admins#update", constraints: { id: /[0-9]+/ }
      delete "admins/:id", to: "admins#destroy", constraints: { id: /[0-9]+/ }
      post "admins/invite", to: "admins#invite"

      # Members management (admin only)
      get "members-management", to: "members_management#index"
      get "members-management/:id", to: "members_management#show", constraints: { id: /[0-9]+/ }
      patch "members-management/:id/status", to: "members_management#update_status", constraints: { id: /[0-9]+/ }
      patch "members-management/:id/_fields", to: "members_management#update_official_fields", constraints: { id: /[0-9]+/ }
      patch "members-management/:id/force-password-reset", to: "members_management#force_password_reset", constraints: { id: /[0-9]+/ }
      delete "members-management/:id", to: "members_management#destroy", constraints: { id: /[0-9]+/ }

      # Attendance
      get "attendance", to: "attendance#index"
      get "attendance/today", to: "attendance#today"
      get "attendance/summary", to: "attendance#summary"
      get "attendance/daily-summary", to: "attendance#daily_summary"
      get "attendance/occupancy", to: "attendance#current_occupancy"
      get "attendance/analytics", to: "attendance#analytics"
      post "attendance", to: "attendance#create"
      get "attendance/:id", to: "attendance#show", constraints: { id: /[0-9]+/ }
      patch "attendance/:id/logout", to: "attendance#record_logout", constraints: { id: /[0-9]+/ }

      # Biometric device API
      post "biometric/authenticate-device", to: "biometric#authenticate_device"
      post "biometric/enroll-fingerprint", to: "biometric#enroll_fingerprint"
      get "biometric/enrollments", to: "biometric#list_enrollments"
      delete "biometric/enrollments/:id", to: "biometric#remove_enrollment", constraints: { id: /[0-9]+/ }
      get "biometric/status", to: "biometric#device_status"
      post "biometric/submit-event", to: "biometric#submit_event"
      post "biometric/sync-events", to: "biometric#sync_events"

      # Messages and communication
      get "messages", to: "messages#index"
      get "messages/pinned", to: "messages#pinned_messages"
      get "messages/search", to: "messages#search"
      get "messages/:id", to: "messages#show", constraints: { id: /[0-9]+/ }
      post "messages", to: "messages#create"
      patch "messages/:id", to: "messages#edit", constraints: { id: /[0-9]+/ }
      delete "messages/:id", to: "messages#delete", constraints: { id: /[0-9]+/ }
      post "messages/:id/reply", to: "messages#reply", constraints: { id: /[0-9]+/ }
      get "messages/:id/replies", to: "messages#get_replies", constraints: { id: /[0-9]+/ }
      post "messages/:id/forward", to: "messages#forward", constraints: { id: /[0-9]+/ }
      post "messages/:id/react", to: "messages#react", constraints: { id: /[0-9]+/ }
      delete "messages/:id/react", to: "messages#remove_reaction", constraints: { id: /[0-9]+/ }
      get "messages/:id/reactions", to: "messages#get_reactions", constraints: { id: /[0-9]+/ }
      post "messages/:id/pin", to: "messages#pin", constraints: { id: /[0-9]+/ }
      delete "messages/:id/pin", to: "messages#unpin", constraints: { id: /[0-9]+/ }
      get "messages/:id/receipts", to: "messages#get_receipts", constraints: { id: /[0-9]+/ }
      post "messages/:id/star", to: "messages#star", constraints: { id: /[0-9]+/ }
      delete "messages/:id/star", to: "messages#unstar", constraints: { id: /[0-9]+/ }
      get "messages/:id/starred", to: "messages#get_starred", constraints: { id: /[0-9]+/ }

      # Notifications
      get "notifications", to: "notifications#index"
      get "notifications/unread-count", to: "notifications#unread_count"
      get "notifications/history", to: "notifications#history"
      get "notifications/:id", to: "notifications#show", constraints: { id: /[0-9]+/ }
      post "notifications/read", to: "notifications#mark_read"
      post "notifications/read-all", to: "notifications#mark_all_read"
      delete "notifications/:id", to: "notifications#destroy", constraints: { id: /[0-9]+/ }

      # Announcements
      get "announcements", to: "announcements#index"
      get "announcements/:id", to: "announcements#show", constraints: { id: /[0-9]+/ }
      post "announcements", to: "announcements#create"
      patch "announcements/:id", to: "announcements#update", constraints: { id: /[0-9]+/ }
      post "announcements/:id/publish", to: "announcements#publish", constraints: { id: /[0-9]+/ }
      post "announcements/:id/unpublish", to: "announcements#unpublish", constraints: { id: /[0-9]+/ }
      post "announcements/:id/toggle-pin", to: "announcements#toggle_pin", constraints: { id: /[0-9]+/ }
      delete "announcements/:id", to: "announcements#destroy", constraints: { id: /[0-9]+/ }
      get "announcements/:id/read-status", to: "announcements#read_status", constraints: { id: /[0-9]+/ }

      # Events
      get "events", to: "events#index"
      get "events/:id", to: "events#show", constraints: { id: /[0-9]+/ }
      post "events", to: "events#create"
      patch "events/:id", to: "events#update", constraints: { id: /[0-9]+/ }
      delete "events/:id", to: "events#destroy", constraints: { id: /[0-9]+/ }

      # Tasks
      get "tasks", to: "tasks#index"
      get "tasks/:id", to: "tasks#show", constraints: { id: /[0-9]+/ }
      post "tasks", to: "tasks#create"
      patch "tasks/:id", to: "tasks#update", constraints: { id: /[0-9]+/ }
      delete "tasks/:id", to: "tasks#destroy", constraints: { id: /[0-9]+/ }
      post "tasks/:id/submit", to: "tasks#submit", constraints: { id: /[0-9]+/ }
      post "tasks/:id/verify", to: "tasks#verify_submission", constraints: { id: /[0-9]+/ }

      # Inventory
      get "inventory", to: "inventory#index"
      get "inventory/stats", to: "inventory#stats"
      get "inventory/:id", to: "inventory#show", constraints: { id: /[0-9]+/ }
      post "inventory", to: "inventory#create"
      patch "inventory/:id", to: "inventory#update", constraints: { id: /[0-9]+/ }
      delete "inventory/:id", to: "inventory#destroy", constraints: { id: /[0-9]+/ }
      post "inventory/import", to: "inventory#import"

      # Borrowing
      get "borrowing/requests", to: "borrowing#index"
      get "borrowing/admin/requests", to: "borrowing#admin_index"
      get "borrowing/requests/:id", to: "borrowing#show", constraints: { id: /[0-9]+/ }
      post "borrowing/requests", to: "borrowing#create"
      post "borrowing/requests/:id/approve", to: "borrowing#approve", constraints: { id: /[0-9]+/ }
      post "borrowing/requests/:id/reject", to: "borrowing#reject", constraints: { id: /[0-9]+/ }
      post "borrowing/extension-request", to: "borrowing#request_extension"

      # Returns
      get "returns/requests", to: "returns#index"
      get "returns/admin/requests", to: "returns#admin_index"
      post "returns/request", to: "returns#create"
      get "returns/:id", to: "returns#show", constraints: { id: /[0-9]+/ }
      post "returns/:id/confirm", to: "returns#confirm_return", constraints: { id: /[0-9]+/ }
      post "returns/:id/reject", to: "returns#reject", constraints: { id: /[0-9]+/ }

      # Contact
      post "contact/messages", to: "contact#create"
      get "contact/messages", to: "contact#index"
      get "contact/messages/:id", to: "contact#show", constraints: { id: /[0-9]+/ }
      patch "contact/messages/:id/status", to: "contact#update_status", constraints: { id: /[0-9]+/ }
      delete "contact/messages/:id", to: "contact#destroy", constraints: { id: /[0-9]+/ }
      post "contact/update-email/request", to: "contact#request_contact_email_change"
      post "contact/update-email/verify", to: "contact#verify_contact_email_change"

      # FAQ and Help
      get "faqs", to: "faqs#index"
      get "faqs/categories", to: "faqs#categories"
      get "faqs/:slug", to: "faqs#show_by_slug"
      post "faqs/categories", to: "faqs#create_category"
      post "faqs", to: "faqs#create"
      patch "faqs/:id", to: "faqs#update", constraints: { id: /[0-9]+/ }
      delete "faqs/:id", to: "faqs#destroy", constraints: { id: /[0-9]+/ }
      patch "faqs/reorder", to: "faqs#reorder"

      # Analytics and Reporting (admin only)
      get "analytics/dashboard", to: "analytics#dashboard"
      get "analytics/overview", to: "analytics#overview"
      get "analytics/members", to: "analytics#members"
      get "analytics/attendance", to: "analytics#attendance"
      get "analytics/inventory", to: "analytics#inventory_stats"
      get "analytics/tasks", to: "analytics#task_stats"
      get "analytics/announcements", to: "analytics#announcement_stats"
      get "analytics/notifications", to: "analytics#notification_stats"
      get "analytics/contact", to: "analytics#contact_stats"

      # Audit Logs (primary admin only)
      get "audit-logs", to: "audit_logs#index"
      get "audit-logs/summary", to: "audit_logs#summary"
      get "audit-logs/actions", to: "audit_logs#actions"
      get "audit-logs/:id", to: "audit_logs#show", constraints: { id: /[0-9]+/ }
      get "audit-logs/resource/:resource_type/:resource_id", to: "audit_logs#for_resource", constraints: { resource_type: /[a-zA-Z]+/, resource_id: /[0-9]+/ }
      get "audit-logs/user/:user_id", to: "audit_logs#for_user", constraints: { user_id: /[0-9]+/ }
    end
  end
end
