class MessageStar < ApplicationRecord
  belongs_to :group_message
  belongs_to :user

  validates :user_id, uniqueness: { scope: :group_message_id, message: "already starred this message" }

  scope :for_message, ->(message_id) { where(group_message_id: message_id) }
end
