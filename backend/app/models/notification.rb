class Notification < ApplicationRecord
  belongs_to :recipient, class_name: "Member"
  belongs_to :sender, class_name: "Member", optional: true
  belongs_to :notifiable, polymorphic: true, optional: true

  validates :type, presence: true

  scope :unread, -> { where(read_at: nil) }
  scope :read, -> { where.not(read_at: nil) }
  scope :for_member, ->(member_id) { where(recipient_id: member_id) }
  scope :for_resource, ->(resource_type, resource_id) { where(notifiable_type: resource_type, notifiable_id: resource_id) }
  scope :recent, -> { order(created_at: :desc) }
end
