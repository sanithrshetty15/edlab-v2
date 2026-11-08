class AttendanceLog < ApplicationRecord
  belongs_to :member
  belongs_to :period, class_name: "Semester", optional: true
  belongs_to :device_event, optional: true
  has_many :logout_records, class_name: "AttendanceLog", foreign_key: "login_log_id", dependent: :nullify

  validates :member_id, presence: true
  validates :login_at, presence: true
  validates :status, inclusion: { in: %w[login logout unknown] }

  scope :for_today, -> { where(login_at: Time.current.beginning_of_day..Time.current.end_of_day) }
  scope :for_date, ->(date) { where(login_at: date.beginning_of_day..date.end_of_day) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
  scope :by_status, ->(status) { where(status: status) }
  scope :recent, ->(limit = 10) { order(login_at: :desc).limit(limit) }
end
