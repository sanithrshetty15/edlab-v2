class LinkPreview < ApplicationRecord
  has_one :link_preview_result, class_name: "LinkPreviewResult", foreign_key: "link_preview_id", dependent: :destroy
  has_many :messages, class_name: "Message", foreign_key: "link_preview_id", dependent: :nullify

  validates :url, presence: true, uniqueness: true
  validates :title, length: { maximum: 500 }
  validates :description, length: { maximum: 2000 }
  validates :image_url, length: { maximum: 1000 }
  validates :favicon_url, length: { maximum: 500 }

  scope :recent, -> { order(created_at: :desc) }
  scope :cached, -> { where.not(title: nil) }
end
