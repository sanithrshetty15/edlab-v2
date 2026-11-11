class MessagePin < ApplicationRecord
  belongs_to :group_message
  belongs_to :pinned_by, class_name: "User"

  validates :group_message_id, uniqueness: true

  scope :for_message, ->(message_id) { where(group_message_id: message_id) }
end
