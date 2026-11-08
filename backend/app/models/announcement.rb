class Announcement < ApplicationRecord
  has_many :announcement_reads, dependent: :destroy
  has_many :read_members, through: :announcement_reads, source: :member

  validates :title, presence: true, length: { maximum: 255 }
  validates :body, presence: true
  validates :status, inclusion: { in: %w[draft published archived] }
  validates :visibility, inclusion: { in: %w[all members admins] }
  validates :priority, inclusion: { in: %w[low medium high urgent] }, allow_nil: true

  before_validation :generate_slug, on: :create

  scope :published, -> { where(status: "published") }
  scope :for_member, ->(member) { where(visibility: member.admin? ? "all" : "members") }
  scope :pinned_first, -> { order(pinned_at: :desc, created_at: :desc) }
  scope :recent, -> { order(created_at: :desc) }

  def brand_and_model?
    false
  end

  private

  def generate_slug
    self.slug = title.parameterize if slug.blank?
  end
end
