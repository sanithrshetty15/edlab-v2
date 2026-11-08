class Message < ApplicationRecord
  belongs_to :sender, class_name: "Member"
  belongs_to :group_message, optional: true
  belongs_to :link_preview, optional: true

  has_many :message_receipts, dependent: :destroy
  has_many :receipts, through: :message_receipts, source: :member
  has_many :message_reactions, dependent: :destroy
  has_many :reactions, through: :message_reactions, source: :member
  has_many :message_pins, dependent: :destroy
  has_many :pins, through: :message_pins, source: :member
  has_many :message_stars, dependent: :destroy
  has_many :stars, through: :message_stars, source: :member
  has_many :attachments, class_name: "MessageAttachment", dependent: :destroy

  validates :subject, presence: true, length: { maximum: 255 }
  validates :body, presence: true, unless: :has_link_preview?
  validates :receiver_type, inclusion: { in: %w[group user] }
  validates :visibility, inclusion: { in: %w[members admins all] }
  validates :status, inclusion: { in: %w[draft sent delivered read] }

  before_create :set_reference_number

  scope :for_receiver, ->(receiver_type, receiver_id) { where(receiver_type: receiver_type, receiver_id: receiver_id) }
  scope :for_user, ->(member_id) { where(receiver_type: "user", receiver_id: member_id) }
  scope :for_group, ->(group_id) { where(receiver_type: "group", receiver_id: group_id) }
  scope :sent_by, ->(member_id) { where(sender_id: member_id) }
  scope :unread_for, ->(member_id) { joins(:message_receipts).where(message_receipts: { member_id: member_id, read_at: nil }).distinct }
  scope :visible_to, ->(member) { where(visibility: member.admin? ? "all" : "members") }
  scope :recent, -> { order(created_at: :desc) }
  scope :with_replies, -> { where.not(parent_message_id: nil) }
  scope :top_level, -> { where(parent_message_id: nil) }

  def has_link_preview?
    link_preview_id.present?
  end

  def is_reply?
    parent_message_id.present?
  end

  def root_message
    return self unless parent_message_id.present?
    parent = Message.find_by(id: parent_message_id)
    parent ? parent.root_message : self
  end

  private

  def set_reference_number
    self.reference_number = Message.maximum(:reference_number).to_i + 1
  end
end
