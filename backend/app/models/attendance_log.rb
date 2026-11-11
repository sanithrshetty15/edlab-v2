class AttendanceLog < ApplicationRecord
  belongs_to :member
  belongs_to :created_by, class_name: "User", optional: true

  validates :entry_date, presence: true
  validates :entry_time, presence: true

  scope :for_today, -> { where(entry_date: Date.current) }
  scope :for_date, ->(date) { where(entry_date: date) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
  scope :open_sessions, -> { where(exit_time: nil) }
  scope :recent, ->(limit = 10) { order(entry_time: :desc).limit(limit) }

  def duration_minutes
    return nil unless exit_time
    ((exit_time - entry_time) / 60).round
  end
end
