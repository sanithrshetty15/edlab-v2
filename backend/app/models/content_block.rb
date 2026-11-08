class ContentBlock < ApplicationRecord
  belongs_to :author, class_name: "Member", optional: true
  has_many :content_block_reads, class_name: "ContentBlockRead", foreign_key: "content_block_id", dependent: :destroy

  validates :title, presence: true, length: { maximum: 255 }
  validates :body, presence: true
  validates :content_type, inclusion: { in: %w[article guideline policy announcement] }
  validates :visibility, inclusion: { in: %w[public members admins] }

  scope :for_type, ->(type) { where(content_type: type) }
  scope :for_visibility, ->(member) { where(visibility: member.admin? ? "all" : "members") }
  scope :recent, -> { order(created_at: :desc) }
end
