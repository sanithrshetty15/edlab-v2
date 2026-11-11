class BorrowRecord < ApplicationRecord
  belongs_to :member
  belongs_to :inventory_item
  belongs_to :approved_by, class_name: "User", optional: true
  has_one :extension_request, dependent: :destroy

  validates :status, inclusion: { in: %w[pending approved borrowed rejected returned] }
  validates :due_date, presence: true
  validates :quantity, numericality: { greater_than: 0, only_integer: true }

  scope :pending, -> { where(status: "pending") }
  scope :approved, -> { where(status: "approved") }
  scope :active, -> { where(status: %w[approved borrowed]) }
  scope :rejected, -> { where(status: "rejected") }
  scope :returned, -> { where(status: "returned") }
  scope :for_item, ->(item_id) { where(inventory_item_id: item_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
  scope :recent, -> { order(created_at: :desc) }
end
