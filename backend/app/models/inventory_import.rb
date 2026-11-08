class InventoryImport < ApplicationRecord
  has_one_attached :file
  has_many :processed_items, class_name: "InventoryItem", foreign_key: "import_id", dependent: :nullify

  validates :name, presence: true
  validates :status, inclusion: { in: %w[pending processing completed failed] }

  scope :recent, -> { order(created_at: :desc) }
  scope :processed, -> { where(status: "completed") }
  scope :failed, -> { where(status: "failed") }
  scope :pending, -> { where(status: "pending") }
end
