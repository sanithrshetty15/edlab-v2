class ContactMessage < ApplicationRecord
  belongs_to :member

  validates :subject, presence: true, length: { maximum: 255 }
  validates :message, presence: true
  validates :category, presence: true
  validates :status, inclusion: { in: %w[new in_progress resolved closed] }

  scope :pending, -> { where(status: %w[new in_progress]) }
  scope :resolved, -> { where(status: "resolved") }
  scope :recent, -> { order(created_at: :desc) }
end
