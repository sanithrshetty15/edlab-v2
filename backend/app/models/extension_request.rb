class ExtensionRequest < ApplicationRecord
  belongs_to :borrow_record
  belongs_to :member

  validates :requested_return_date, presence: true
  validates :reason, presence: true, length: { maximum: 1000 }
  validates :status, inclusion: { in: %w[pending approved rejected] }

  scope :pending, -> { where(status: "pending") }
  scope :approved, -> { where(status: "approved") }
  scope :rejected, -> { where(status: "rejected") }
  scope :for_borrow, ->(borrow_id) { where(borrow_record_id: borrow_id) }
  scope :recent, -> { order(created_at: :desc) }
end
