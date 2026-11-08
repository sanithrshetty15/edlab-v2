class Event < ApplicationRecord
  has_many :event_attendees, class_name: "EventAttendee", foreign_key: "event_id", dependent: :destroy
  has_many :attendees, through: :event_attendees, source: :member

  validates :title, presence: true, length: { maximum: 255 }
  validates :event_date, presence: true
  validates :status, inclusion: { in: %w[draft scheduled ongoing completed cancelled] }
  validates :event_type, inclusion: { in: %w[lab_seminar workshop hackathon guest_session general_other] }
  validates :location, length: { maximum: 255 }

  scope :upcoming, -> { where(event_type: "event_date", status: "scheduled").order(event_date: :asc) }
  scope :past, -> { where(event_date: ..Time.current).order(event_date: :desc) }
  scope :by_status, ->(status) { where(status: status) }
  scope :by_type, ->(type) { where(event_type: type) }
end
