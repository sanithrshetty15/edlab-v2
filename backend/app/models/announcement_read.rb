class AnnouncementRead < ApplicationRecord
  belongs_to :member
  belongs_to :announcement

  validates :member_id, uniqueness: { scope: :announcement_id, message: "already read this announcement" }

  scope :for_announcement, ->(announcement_id) { where(announcement_id: announcement_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
  scope :unread, -> { where(read_at: nil) }
end
