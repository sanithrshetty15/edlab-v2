class TaskSubmissionStatus < ApplicationRecord
  belongs_to :task
  belongs_to :member
  belongs_to :verified_by, class_name: "User", optional: true

  validates :status, inclusion: { in: %w[pending submitted verified rejected] }
  validates :member_id, uniqueness: { scope: :task_id, message: "already submitted for this task" }

  scope :for_task, ->(task_id) { where(task_id: task_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
  scope :submitted, -> { where(status: "submitted") }
  scope :verified, -> { where(status: "verified") }
  scope :rejected, -> { where(status: "rejected") }
end
