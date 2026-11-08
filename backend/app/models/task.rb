class Task < ApplicationRecord
  has_many :task_submission_statuses, dependent: :destroy
  has_many :task_targets, dependent: :destroy
  has_many :targeted_members, through: :task_targets, source: :member

  validates :title, presence: true, length: { maximum: 255 }
  validates :description, presence: true
  validates :assigned_to_type, inclusion: { in: %w[all specific] }
  validates :status, inclusion: { in: %w[draft open in_progress verifying completed archived] }
  validates :deadline, presence: true

  scope :for_member, ->(member_id) { joins(:task_targets).where(task_targets: { member_id: member_id }) }
  scope :open, -> { where(status: "open") }
  scope :in_progress, -> { where(status: "in_progress") }
  scope :pending_verification, -> { where(status: "verifying") }
  scope :by_deadline, -> { order(deadline: :asc) }
  scope :recent, -> { order(created_at: :desc) }
end
