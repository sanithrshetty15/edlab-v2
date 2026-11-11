class Announcement < ApplicationRecord
  belongs_to :created_by, class_name: "User"
  has_many :announcement_reads, dependent: :destroy
  has_many :readers, through: :announcement_reads, source: :user

  validates :title, presence: true, length: { maximum: 255 }
  validates :content, presence: true

  scope :published, -> { where.not(published_at: nil) }
  scope :visible, -> { where(deleted_at: nil) }
  scope :pinned_first, -> { order(is_pinned: :desc, published_at: :desc, created_at: :desc) }
  scope :recent, -> { order(created_at: :desc) }
end
