class Notification < ApplicationRecord
  belongs_to :user

  validates :title, presence: true
  validates :notification_type, presence: true

  scope :unread, -> { where(is_read: false) }
  scope :read, -> { where(is_read: true) }
  scope :for_user, ->(user_id) { where(user_id: user_id) }
  scope :recent, -> { order(created_at: :desc) }
end
