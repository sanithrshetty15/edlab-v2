class DeviceEvent < ApplicationRecord
  belongs_to :device_registration
  has_one :processed_attendance_log, class_name: "AttendanceLog", foreign_key: "device_event_id"

  validates :event_type, inclusion: { in: %w[LOGIN LOGOUT] }
  validates :member_identifier, presence: true

  scope :unprocessed, -> { where(processed: false).order(created_at: :asc) }
  scope :processed, -> { where(processed: true) }
  scope :by_event_type, ->(type) { where(event_type: type) }
  scope :by_member_identifier, ->(identifier) { where(member_identifier: identifier) }
end
