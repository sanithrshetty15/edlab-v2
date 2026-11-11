class ReturnRequest < ApplicationRecord
  belongs_to :borrow_record
  belongs_to :member
  belongs_to :confirmed_by, class_name: "User", optional: true

  validates :status, inclusion: { in: %w[pending confirmed rejected] }

  scope :pending, -> { where(status: "pending") }
  scope :confirmed, -> { where(status: "confirmed") }
  scope :rejected, -> { where(status: "rejected") }
  scope :for_borrow, ->(borrow_id) { where(borrow_record_id: borrow_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
  scope :recent, -> { order(created_at: :desc) }
end
