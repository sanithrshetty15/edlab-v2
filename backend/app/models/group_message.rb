class GroupMessage < ApplicationRecord
  belongs_to :user
  belongs_to :reply_to, class_name: "GroupMessage", optional: true
  belongs_to :forwarded_from, class_name: "GroupMessage", optional: true
  has_many :attachments, class_name: "MessageAttachment", foreign_key: "group_message_id", dependent: :destroy
  has_many :receipts, class_name: "MessageReceipt", foreign_key: "group_message_id", dependent: :destroy
  has_many :reactions, class_name: "MessageReaction", foreign_key: "group_message_id", dependent: :destroy
  has_many :pins, class_name: "MessagePin", foreign_key: "group_message_id", dependent: :destroy
  has_many :stars, class_name: "MessageStar", foreign_key: "group_message_id", dependent: :destroy
  has_one :link_preview, class_name: "LinkPreview", foreign_key: "group_message_id", dependent: :destroy

  validates :content, presence: true, unless: -> { message_type == "media" }

  scope :visible, -> { where(is_deleted: false) }
  scope :pinned, -> { where(is_pinned: true, is_deleted: false) }
  scope :recent, -> { order(created_at: :desc) }
end
