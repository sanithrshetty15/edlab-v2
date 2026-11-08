class TaskTarget < ApplicationRecord
  belongs_to :task
  belongs_to :member

  validates :member_id, uniqueness: { scope: :task_id, message: "already targeted for this task" }

  scope :for_task, ->(task_id) { where(task_id: task_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
end
