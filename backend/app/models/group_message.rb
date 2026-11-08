class GroupMessage < ApplicationRecord
  belongs_to :sender, class_name: "Member"
  has_many :group_message_receipts, class_name: "MessageReceipt", foreign_key: "message_id", dependent: :destroy
  has_many :group_message_reactions, class_name: "MessageReaction", foreign_key: "message_id", dependent: :destroy
  has_many :group_message_pins, class_name: "MessagePin", foreign_key: "message_id", dependent: :destroy
  has_many :group_message_stars, class_name: "MessageStar", foreign_key: "message_id", dependent: :destroy

  validates :subject, presence: true, length: { maximum: 255 }
  validates :body, presence: true
  validates :receiver_type, inclusion: { in: %w[group user] }
  validates :visibility, inclusion: { in: %w[members admins all] }

  scope :for_group, ->(group_id) { where(receiver_type: "group", group_id: group_id) }
  scope :for_user, ->(member_id) { where(receiver_type: "user", user_id: member_id) }
  scope :unread_for, ->(member_id) { where.not(id: MessageReceipt.where(member_id: member_id, read_at: nil).pluck(:message_id)) }
  scope :visible_to, ->(member) { where(visibility: member.admin? ? "all" : "members") }
end
