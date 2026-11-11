class Task < ApplicationRecord
  belongs_to :created_by, class_name: "User"
  has_many :task_submission_statuses, dependent: :destroy
  has_many :task_targets, dependent: :destroy
  has_many :submissions, through: :task_submission_statuses, source: :member

  validates :title, presence: true, length: { maximum: 255 }
  validates :task_type, presence: true
  validates :deadline, presence: true

  scope :visible, -> { where(deleted_at: nil) }
  scope :open, -> { where("deadline >= ?", Date.current) }
  scope :overdue, -> { where("deadline < ?", Date.current) }
  scope :by_deadline, -> { order(deadline: :asc) }
  scope :recent, -> { order(created_at: :desc) }
end
