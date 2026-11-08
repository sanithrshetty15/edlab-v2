class ReturnRequest < ApplicationRecord
  belongs_to :borrow_record
  belongs_to :member

  validates :status, inclusion: { in: %w[pending approved rejected] }
  validates :expected_return_date, presence: true, if: :pending?

  scope :pending, -> { where(status: "pending") }
  scope :approved, -> { where(status: "approved") }
  scope :for_borrow, ->(borrow_id) { where(borrow_record_id: borrow_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
  scope :recent, -> { order(created_at: :desc) }
end
