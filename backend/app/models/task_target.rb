class TaskTarget < ApplicationRecord
  belongs_to :task

  validates :academic_year, presence: true, uniqueness: { scope: :task_id, message: "already targeted for this task" }

  scope :for_task, ->(task_id) { where(task_id: task_id) }
end
