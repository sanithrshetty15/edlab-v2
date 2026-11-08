class MessageStar < ApplicationRecord
  belongs_to :message
  belongs_to :member

  validates :member_id, uniqueness: { scope: :message_id, message: "already starred this message" }

  scope :for_message, ->(message_id) { where(message_id: message_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
end
