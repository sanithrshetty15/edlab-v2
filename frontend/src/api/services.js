import { api } from "./client";

const BASE = "/api/v1";

export const authApi = {
  memberLogin: ({ loginType, loginId, password }) =>
    api.post(
      `${BASE}/auth/member-login`,
      { login_type: loginType, login_id: loginId, password },
      { auth: false },
    ),
  adminLogin: ({ email }) =>
    api.post(`${BASE}/auth/admin-login`, { email }, { auth: false }),
  verifyAdminOTP: ({ otpCode, email }) =>
    api.post(`${BASE}/auth/admin-verify-otp`, { otp: otpCode, email }, { auth: false }),
  logout: () => api.delete(`${BASE}/auth/logout`),
  me: () => api.get(`${BASE}/auth/me`),
  changePassword: ({ currentPassword, newPassword, passwordConfirmation }) =>
    api.patch(`${BASE}/auth/change-password`, {
      current_password: currentPassword,
      new_password: newPassword,
      password_confirmation: passwordConfirmation,
    }),
  requestPasswordReset: ({ loginId }) =>
    api.post(
      `${BASE}/auth/password-reset/request`,
      { login_id: loginId, delivery_method: "email" },
      { auth: false },
    ),
  completePasswordReset: ({ loginId, otpCode, newPassword, passwordConfirmation }) =>
    api.post(
      `${BASE}/auth/password-reset/complete`,
      {
        login_id: loginId,
        otp: otpCode,
        new_password: newPassword,
        password_confirmation: passwordConfirmation,
      },
      { auth: false },
    ),
};

export const membersApi = {
  profile: () => api.get(`${BASE}/members/me`),
  updateProfile: (payload) => api.patch(`${BASE}/members/me/profile`, payload),
  acceptRules: () => api.patch(`${BASE}/members/me/accept-rules`, {}),
};

export const membersManagementApi = {
  list: (params) => api.get(`${BASE}/members-management`, { query: params }),
  create: (payload) => api.post(`${BASE}/members-management`, payload),
  get: (id) => api.get(`${BASE}/members-management/${id}`),
  updateStatus: (id, status) =>
    api.patch(`${BASE}/members-management/${id}/status`, { status }),
  updateFields: (id, fields) =>
    api.patch(`${BASE}/members-management/${id}/_fields`, fields),
  forcePasswordReset: (id) =>
    api.patch(`${BASE}/members-management/${id}/force-password-reset`, {}),
  startFingerprintEnrollment: (id, payload = {}) =>
    api.post(`${BASE}/members-management/${id}/fingerprint-enrollment`, payload),
  getFingerprintEnrollment: (id) =>
    api.get(`${BASE}/members-management/${id}/fingerprint-enrollment`),
  getEnrollmentReaders: (id, params) =>
    api.get(`${BASE}/members-management/${id}/fingerprint-enrollment/readers`, {
      query: params,
    }),
  cancelFingerprintEnrollment: (id) =>
    api.delete(`${BASE}/members-management/${id}/fingerprint-enrollment`),
  getRegistrationEmail: (id) =>
    api.get(`${BASE}/members-management/${id}/registration-email`),
  retryRegistrationEmail: (id) =>
    api.post(`${BASE}/members-management/${id}/registration-email/retry`, {}),
  deactivate: (id) => api.delete(`${BASE}/members-management/${id}`),
};

export const attendanceApi = {
  list: (params) => api.get(`${BASE}/attendance`, { query: params }),
  get: (id) => api.get(`${BASE}/attendance/${id}`),
  today: () => api.get(`${BASE}/attendance/today`),
  summary: (params) => api.get(`${BASE}/attendance/summary`, { query: params }),
  dailySummary: (params) => api.get(`${BASE}/attendance/daily-summary`, { query: params }),
  occupancy: () => api.get(`${BASE}/attendance/occupancy`),
  analytics: (params) => api.get(`${BASE}/attendance/analytics`, { query: params }),
  create: (payload) => api.post(`${BASE}/attendance`, payload),
  recordLogout: (id) => api.patch(`${BASE}/attendance/${id}/logout`, {}),
};

export const announcementsApi = {
  list: (params) => api.get(`${BASE}/announcements`, { query: params }),
  get: (id) => api.get(`${BASE}/announcements/${id}`),
  create: (payload) => api.post(`${BASE}/announcements`, payload),
  update: (id, payload) => api.patch(`${BASE}/announcements/${id}`, payload),
  publish: (id, payload = {}) => api.post(`${BASE}/announcements/${id}/publish`, payload),
  unpublish: (id) => api.post(`${BASE}/announcements/${id}/unpublish`, {}),
  togglePin: (id) => api.post(`${BASE}/announcements/${id}/toggle-pin`, {}),
  remove: (id) => api.delete(`${BASE}/announcements/${id}`),
  readStatus: (id) => api.get(`${BASE}/announcements/${id}/read-status`),
};

export const eventsApi = {
  list: (params) => api.get(`${BASE}/events`, { query: params }),
  get: (id) => api.get(`${BASE}/events/${id}`),
  create: (payload) => api.post(`${BASE}/events`, payload),
  update: (id, payload) => api.patch(`${BASE}/events/${id}`, payload),
  remove: (id) => api.delete(`${BASE}/events/${id}`),
};

export const messagesApi = {
  list: (params) => api.get(`${BASE}/messages`, { query: params }),
  get: (id) => api.get(`${BASE}/messages/${id}`),
  create: (payload) => api.post(`${BASE}/messages`, payload),
  reply: (id, payload) => api.post(`${BASE}/messages/${id}/reply`, payload),
  edit: (id, payload) => api.patch(`${BASE}/messages/${id}`, payload),
  remove: (id) => api.delete(`${BASE}/messages/${id}`),
  pinned: () => api.get(`${BASE}/messages/pinned`),
  search: (q) => api.get(`${BASE}/messages/search`, { query: { q } }),
  pin: (id) => api.post(`${BASE}/messages/${id}/pin`, {}),
  unpin: (id) => api.delete(`${BASE}/messages/${id}/pin`),
};

export const tasksApi = {
  list: (params) => api.get(`${BASE}/tasks`, { query: params }),
  get: (id) => api.get(`${BASE}/tasks/${id}`),
  create: (payload) => api.post(`${BASE}/tasks`, payload),
  update: (id, payload) => api.patch(`${BASE}/tasks/${id}`, payload),
  remove: (id) => api.delete(`${BASE}/tasks/${id}`),
  submit: (id) => api.post(`${BASE}/tasks/${id}/submit`, {}),
  verify: (id, payload) => api.post(`${BASE}/tasks/${id}/verify`, payload),
};

export const inventoryApi = {
  list: (params) => api.get(`${BASE}/inventory`, { query: params }),
  get: (id) => api.get(`${BASE}/inventory/${id}`),
  stats: () => api.get(`${BASE}/inventory/stats`),
  create: (payload) => api.post(`${BASE}/inventory`, payload),
  update: (id, payload) => api.patch(`${BASE}/inventory/${id}`, payload),
  remove: (id) => api.delete(`${BASE}/inventory/${id}`),
};

export const borrowingApi = {
  list: (params) => api.get(`${BASE}/borrowing/requests`, { query: params }),
  adminList: (params) => api.get(`${BASE}/borrowing/admin/requests`, { query: params }),
  get: (id) => api.get(`${BASE}/borrowing/requests/${id}`),
  create: (payload) => api.post(`${BASE}/borrowing/requests`, payload),
  approve: (id, payload = {}) => api.post(`${BASE}/borrowing/requests/${id}/approve`, payload),
  reject: (id, payload) => api.post(`${BASE}/borrowing/requests/${id}/reject`, payload),
  requestExtension: (payload) => api.post(`${BASE}/borrowing/extension-request`, payload),
};

export const returnsApi = {
  list: (params) => api.get(`${BASE}/returns/requests`, { query: params }),
  adminList: (params) => api.get(`${BASE}/returns/admin/requests`, { query: params }),
  create: (payload) => api.post(`${BASE}/returns/request`, payload),
  confirm: (id, payload = {}) => api.post(`${BASE}/returns/${id}/confirm`, payload),
  reject: (id, payload) => api.post(`${BASE}/returns/${id}/reject`, payload),
};

export const analyticsApi = {
  dashboard: () => api.get(`${BASE}/analytics/dashboard`),
  overview: () => api.get(`${BASE}/analytics/overview`),
  members: () => api.get(`${BASE}/analytics/members`),
  attendance: (params) => api.get(`${BASE}/analytics/attendance`, { query: params }),
  inventory: () => api.get(`${BASE}/analytics/inventory`),
  tasks: () => api.get(`${BASE}/analytics/tasks`),
  announcements: () => api.get(`${BASE}/analytics/announcements`),
  notifications: () => api.get(`${BASE}/analytics/notifications`),
  contact: () => api.get(`${BASE}/analytics/contact`),
};

export const notificationsApi = {
  list: (params) => api.get(`${BASE}/notifications`, { query: params }),
  unreadCount: () => api.get(`${BASE}/notifications/unread-count`),
  markRead: (ids) => api.post(`${BASE}/notifications/read`, { ids }),
  markAllRead: () => api.post(`${BASE}/notifications/read-all`, {}),
  remove: (id) => api.delete(`${BASE}/notifications/${id}`),
};
