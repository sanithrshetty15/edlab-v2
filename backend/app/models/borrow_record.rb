class BorrowRecord < ApplicationRecord
  belongs_to :member
  belongs_to :inventory_item

  validates :quantity, presence: true, numericality: { greater_than: 0, only_integer: true }
  validates :purpose, presence: true, length: { maximum: 1000 }
  validates :status, inclusion: { in: %w[pending approved rejected returned] }
  validates :estimated_return_date, presence: true

  scope :pending, -> { where(status: "pending") }
  scope :approved, -> { where(status: "approved") }
  scope :rejected, -> { where(status: "rejected") }
  scope :returned, -> { where(status: "returned") }
  scope :for_item, ->(item_id) { where(inventory_item_id: item_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
  scope :active, -> { where(status: "approved") }
  scope :recent, -> { order(created_at: :desc) }
end
