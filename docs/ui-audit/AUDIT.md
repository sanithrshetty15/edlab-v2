# EdwinsLab Admin Console UI Audit

**Audit date:** 2026-11-26  
**Scope:** Visual and usability review before the UI polish pass  
**Method:** Code audit of the React routes and shared UI styles. Playwright and a Chromium binary are installed, but authenticated admin-page screenshots could not be captured in this audit because the local app requires a live authenticated session and backend data; no screenshot files are claimed here.

## Severity

- **P0:** Blocks rendering or prevents core navigation.
- **P1:** Major usability/accessibility issue or likely collision on a required viewport.
- **P2:** Noticeable polish, consistency, or responsive issue.
- **P3:** Minor refinement.

## Global findings

- **P0 — Build blocker:** `AttendancePage.jsx` contained an invalid nested JSX expression in the records branch. The records chain is now wrapped in a fragment.
- **P0 — Build blocker:** The `ReaderStatusPanel` import path was invalid after the component moved. It now imports `Ui` locally and `MemberEnrollmentDialog` imports the shared panel path.
- **P1 — Lint baseline:** `npm run lint` currently reports 29 errors and 1 warning across `AppPortal.jsx`, `ReaderStatusPanel.jsx`, `AttendancePage.jsx`, `MemberEnrollmentDialog.jsx`, and `MembersPage.jsx`. These include undefined components/variables, unused imports/state, and hook rule violations.
- **P1 — Typography:** Shared admin styles need a systematic pass to enforce the requested minimums: 16px body/table text, 14px secondary text, and 13px labels/badges.
- **P1 — Responsive data:** Admin data tables need explicit mobile card layouts below 768px; essential record information must not depend on horizontal scrolling.
- **P1 — Interaction accessibility:** Interactive rows, icon buttons, tabs, dialogs, and menus need a keyboard/focus audit with visible `:focus-visible` states and 44px touch targets.
- **P2 — Loading consistency:** Loading behavior is split across page-specific loaders and panel states. Consolidate on the lime spinner/three-dot loader and verify reduced-motion behavior.
- **P2 — Feedback consistency:** Empty, error-with-retry, success, toast, and pagination states should use one shared visual language.
- **P2 — Navigation consistency:** Verify there is exactly one Attendance sidebar item and that `/app/live-attendance` redirects without leaving a duplicate navigation affordance.

## Page-by-page findings

### Login

- **P1:** Authentication form and OTP/error feedback require a keyboard and focus-order check.
- **P2:** Verify labels, error placement, loading state, and contrast at 360px.
- **P2:** OTP input and submit controls should retain 44px touch targets.

### Forgot password

- **P1:** Verify the recovery/OTP flow has clear inline errors and success feedback without layout shift.
- **P2:** Check form spacing and readable secondary copy at 360px.

### Dashboard

- **P1:** Stat tiles need stable reserved space while data loads to prevent layout shift.
- **P1:** Numbers and labels need the requested readable scale and stronger secondary-text contrast.
- **P2:** Verify empty/error states for each dashboard data section.

### Members

- **P1:** Convert the member list to cards below 768px; preserve all essential identity/status/actions.
- **P1:** Member row/card click should expose a clearly focusable detail drawer with an accessible close action.
- **P1:** Filter controls must wrap cleanly and maintain touch spacing at 360px.
- **P2:** Confirm the year filter is limited to Year 1 through Year 4.
- **P2:** Resolve currently reported unused OTP handlers and missing `ArrowUpIcon` lint errors before page polish is considered complete.

### Attendance — live feed

- **P1:** Live feed rows need explicit Entry, Exit, and Not recognized text badges; colour must not be the only signal.
- **P1:** Feed rows need a mobile card treatment below 768px.
- **P1:** Any clickable event/member row needs keyboard semantics and a visible focus ring.
- **P2:** Verify offline/loading/error/empty states use the shared loader and retry treatment.

### Attendance — records

- **P0:** Invalid nested JSX was fixed before audit work.
- **P1:** The table needs a mobile card layout below 768px rather than relying on horizontal overflow.
- **P1:** The record detail action should be consistently available from every row and keyboard accessible.
- **P2:** Verify open-session status has text plus a non-colour indicator and that pagination remains usable at 360px.
- **P2:** Resolve the current lint errors for undefined `TextInput`, `DefinitionList`, `PageHeader`, `Pagination`, `formatTime`, and unused state/imports.

### Devices

- **P1:** Device status needs a clear online/offline text treatment and a calm reduced-motion status animation.
- **P1:** Device list/table needs a mobile card layout.
- **P2:** Verify error-with-retry and empty states.

### Announcements

- **P1:** Verify announcement cards/list items do not collide at 360px and preserve readable body text.
- **P2:** Standardize empty, error, and action feedback states.

### Events

- **P1:** Verify event cards stack cleanly at 360px and keep primary actions visible.
- **P2:** Dates, status badges, and actions need consistent hierarchy and accessible labels.

### Tasks

- **P1:** Verify task rows/cards remain usable without horizontal scrolling on mobile.
- **P2:** Status and completion indicators need text in addition to colour.

### Messages

- **P1:** Verify message list/detail transitions, focus management, and readable message text on narrow screens.
- **P2:** Empty and error states should provide a clear next action.

### Inventory

- **P1:** Inventory data requires a mobile card representation below 768px.
- **P2:** Availability/status needs a text label and consistent badge treatment.

### Borrowing

- **P1:** Borrowing/request rows require mobile cards and clear action grouping.
- **P2:** Verify loading, empty, error, and success feedback states.

### Analytics

- **P1:** Charts and stat content need a narrow-screen layout review at 360px and 768px.
- **P1:** Chart legends/tooltips must not rely on colour alone and need accessible names.
- **P2:** Reserve chart dimensions to avoid layout shift while data loads.

### Notifications

- **P1:** Verify unread state has a text/semantic indicator in addition to colour or a dot.
- **P2:** Notification rows need comfortable touch targets and clear empty state copy.

### Profile

- **P1:** Avatar area needs stable space for the upload control being added by the separate task.
- **P2:** Verify form labels, focus states, validation, and success feedback.

### Settings

- **P1:** Verify settings sections, controls, and destructive actions remain separated and readable at 360px.
- **P2:** Standardize inputs, selects, toggles, and save feedback with shared components.

## Audit limitations

- No authenticated browser session was available, so no before screenshots were saved and no visual claim is based on a rendered admin session.
- The 21st MCP endpoint is configured in Claude’s config but is not available as a callable tool in this OpenCode session. No 21st component or code was used.
- The audit intentionally did not change backend code, API contracts, attendance terminal endpoints, or business logic.
