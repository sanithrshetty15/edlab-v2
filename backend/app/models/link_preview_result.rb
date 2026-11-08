class LinkPreviewResult < ApplicationRecord
  belongs_to :link_preview

  validates :status, inclusion: { in: %w[pending success error] }
  validates :error_message, length: { maximum: 1000 }

  scope :pending, -> { where(status: "pending") }
  scope :success, -> { where(status: "success") }
  scope :error, -> { where(status: "error") }
end
