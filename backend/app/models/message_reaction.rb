class MessageReaction < ApplicationRecord
  belongs_to :group_message
  belongs_to :user

  validates :emoji, inclusion: { in: %w[👍 ❤️ 😂 😮 😢 🙏] }
  validates :user_id, uniqueness: { scope: [:group_message_id, :emoji], message: "already reacted to this message" }

  scope :for_message, ->(message_id) { where(group_message_id: message_id) }
end
