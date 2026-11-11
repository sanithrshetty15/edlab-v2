class AnnouncementRead < ApplicationRecord
  belongs_to :announcement
  belongs_to :user

  validates :user_id, uniqueness: { scope: :announcement_id, message: "already read this announcement" }

  scope :for_announcement, ->(announcement_id) { where(announcement_id: announcement_id) }
  scope :for_user, ->(user_id) { where(user_id: user_id) }
  scope :unread, -> { where(read_at: nil) }
end
