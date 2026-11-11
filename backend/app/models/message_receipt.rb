class MessageReceipt < ApplicationRecord
  belongs_to :group_message
  belongs_to :user

  validates :user_id, uniqueness: { scope: :group_message_id, message: "already has a receipt for this message" }

  scope :for_message, ->(message_id) { where(group_message_id: message_id) }
  scope :for_user, ->(user_id) { where(user_id: user_id) }
  scope :read, -> { where.not(read_at: nil) }
  scope :unread, -> { where(read_at: nil) }
end
