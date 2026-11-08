class AdminTransfer < ApplicationRecord
  belongs_to :from_admin, class_name: "Admin"
  belongs_to :to_admin, class_name: "Admin"
  belongs_to :performed_by, class_name: "Admin"

  validates :status, inclusion: { in: %w[pending approved rejected] }
  validates :purpose, presence: true, length: { maximum: 1000 }

  scope :pending, -> { where(status: "pending") }
  scope :approved, -> { where(status: "approved") }
  scope :for_admin, ->(admin_id) { where(from_admin_id: admin_id).or(where(to_admin_id: admin_id)) }
end
