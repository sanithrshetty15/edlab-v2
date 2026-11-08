class ContactMessage < ApplicationRecord
  belongs_to :member, optional: true
  belongs_to :sender, class_name: "Member", optional: true

  validates :subject, presence: true, length: { maximum: 255 }
  validates :body, presence: true
  validates :email, presence: true, format: { with: URI::MailTo::EMAIL_REGEXP }, if: -> { member_id.blank? || sender_id.blank? }
  validates :status, inclusion: { in: %w[pending reviewed resolved] }

  scope :pending, -> { where(status: "pending") }
  scope :reviewed, -> { where(status: "reviewed") }
  scope :resolved, -> { where(status: "resolved") }
  scope :recent, -> { order(created_at: :desc) }
end
