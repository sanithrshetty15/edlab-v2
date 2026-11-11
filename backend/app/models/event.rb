class Event < ApplicationRecord
  belongs_to :created_by, class_name: "User"

  validates :title, presence: true, length: { maximum: 255 }
  validates :event_type, presence: true, inclusion: { in: %w[workshop seminar talk hackathon training other] }
  validates :date, presence: true
  validates :start_time, presence: true
  validates :end_time, presence: true

  scope :visible, -> { where(deleted_at: nil) }
  scope :upcoming, -> { where("date >= ?", Date.current).order(date: :asc, start_time: :asc) }
  scope :past, -> { where("date < ?", Date.current).order(date: :desc, start_time: :desc) }
  scope :by_status, ->(status) { where(event_type: status) }
  scope :recent, -> { order(created_at: :desc) }
end
