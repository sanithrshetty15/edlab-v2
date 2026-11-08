# Edwins Lab Backend — Development Progress

**Project:** Edwins Lab — College Open Source Technology Lab Management Platform  
**Backend Location:** `/home/sanith/Desktop/edwinslab/backend`  
**Date Started:** November 4, 2026  
**Backend Engineer:** Buffy (AI Assistant)

---

## Current Status Summary

### ✅ COMPLETED — Foundation

| Component | Status | Details |
|-----------|--------|---------|
| Rails API Setup | ✅ Complete | Rails 8.1.4, API mode, PostgreSQL |
| Database Schema | ✅ Complete | 44 tables with proper relationships |
| User Model | ✅ Complete | `has_secure_password`, roles, sessions |
| Member Model | ✅ Complete | USN/roll_number, academic info, profile fields |
| Admin Model | ✅ Complete | Primary/secondary admins, user linkage |
| Session Model | ✅ Complete | Hashed tokens, expiration, IP tracking |
| OTP System | ✅ Complete | HMAC-SHA256, 5-min expiry, rate limiting |
| Gmail SMTP | ✅ Complete | Configured via ENV variables |
| Auth Controller | ✅ Complete | Member login, admin login, logout, me, change password |
| Password Reset | ✅ Complete | Request/complete flow with OTP |
| Health Endpoint | ✅ Complete | `/api/v1/health` |
| CORS Configuration | ✅ Complete | localhost:5173 allowed |
| Audit Logs Table | ✅ Complete | Schema exists, logging not yet implemented |

### 🔄 IN PROGRESS — Phase 1: Membership and Registration

| Component | Status | Notes |
|-----------|--------|-------|
| Member Registration | ⬜ Not Started | Need to build registration endpoint |
| Registration Validation | ⬜ Not Started | USN, roll_number, email, phone validations |
| Profile Management | ⬜ Not Started | Editable vs controlled fields |
| Activation/Deactivation | ⬜ Not Started | Status management |

### ⬜ PENDING Phases

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 2 | Administration and Authorization | ⬜ Not Started |
| Phase 3 | Attendance Management | ⬜ Not Started |
| Phase 4 | Biometric Attendance Integration | ⬜ Not Started |
| Phase 5 | Communication System (Messages) | ⬜ Not Started |
| Phase 6 | Announcements | ⬜ Not Started |
| Phase 7 | Events and Tasks | ⬜ Not Started |
| Phase 8 | Inventory Management | ⬜ Not Started |
| Phase 9 | Notifications | ⬜ Not Started |
| Phase 10 | Contact, Help and FAQ | ⬜ Not Started |
| Phase 11 | Analytics and Reporting | ⬜ Not Started |
| Phase 12 | Audit Logging | ⬜ Not Started |

---

## Database Schema Overview (44 tables)

### Core Tables
- `users` — Authentication, roles, password
- `members` — Member profile, academic info, status
- `admins` — Admin records, primary/secondary
- `sessions` — Server-side session storage

### Authentication & Security
- `otp_codes` — OTP with HMAC digest, attempts, expiry
- `audit_logs` — Action tracking

### Attendance
- `attendance_logs` — Login/logout records
- `biometric_enrollments` — Fingerprint-to-member mappings
- `device_registrations` — Hardware device auth
- `device_events` — Queued biometric events

### Communication
- `group_messages` — Discussion messages
- `message_attachments` — File/image attachments
- `message_reactions` — Emoji reactions
- `message_receipts` — Read/delivery tracking
- `message_pins` — Pinned messages
- `message_stars` — Starred messages
- `polls` / `poll_options` / `poll_votes` — Polls
- `link_previews` — URL preview data
- `group_settings` — Group permissions

### Content Management
- `announcements` / `announcement_reads` — Announcements
- `events` — Lab events
- `tasks` / `task_targets` / `task_submission_statuses` — Tasks
- `faq_categories` / `faqs` — FAQ system
- `content_blocks` — Reusable content
- `lab_rules_versions` / `lab_rules_acceptances` — Rules tracking

### Inventory
- `inventory_items` — Stock items
- `inventory_imports` — CSV/Excel import tracking
- `borrow_records` — Borrowing transactions
- `return_requests` — Return workflow

### Notifications & Contact
- `notifications` — In-app notifications
- `contact_messages` — Contact form submissions
- `platform_settings` — System settings

### Supporting
- `academic_years` — Academic year reference
- `semesters` — Semester reference

---

## Environment Variables (from `.env`)

```bash
GMAIL_USERNAME=edwinsoslab@gmail.com
GMAIL_APP_PASSWORD=<App Password>
RESEND_API_KEY=<Legacy, not currently used>
OTP_TEST_RECIPIENT=edwinsoslab@gmail.com
```

---

## Key Design Decisions Already Made

1. **Server-side sessions** with hashed tokens (SHA-256), not JWT
2. **OTP with HMAC-SHA256** digest, never storing raw codes
3. **Database-level constraints** for data integrity (unique indexes, check constraints)
4. **Soft deletion** via `deleted_at` timestamps on members and inventory
5. **Idempotency keys** on device events and borrow records
6. **Role-based access** via `users.role` and `admins.is_primary`

---

## What Needs to Be Built

### Immediate Priority (Phase 1)
1. Member registration endpoint with full validation
2. Profile update endpoints (editable fields only)
3. Member activation/deactivation
4. First-password-change flow

### Next Priorities (Phases 2-12)
- Full admin authorization system
- Attendance CRUD and reporting
- Biometric device API
- Discussion/communication system
- Announcements with notifications
- Events and tasks
- Inventory borrowing system
- Notification system
- Contact/FAQ management
- Analytics dashboards
- Comprehensive audit logging

---

*Last updated: November 4, 2026*
