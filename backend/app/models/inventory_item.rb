class InventoryItem < ApplicationRecord
  has_many :borrow_records, dependent: :restrict_with_exception

  validates :name, presence: true, length: { maximum: 255 }
  validates :category, presence: true
  validates :total_quantity, numericality: { greater_than_or_equal_to: 0, only_integer: true }
  validates :available_quantity, numericality: { greater_than_or_equal_to: 0, only_integer: true }

  scope :active, -> { where(is_active: true, deleted_at: nil) }
  scope :by_category, ->(category) { where(category: category) }
  scope :search, ->(query) { where("name ILIKE ? OR description ILIKE ? OR category ILIKE ?", "%#{query}%", "%#{query}%", "%#{query}%") }
end
