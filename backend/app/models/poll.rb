class Poll < ApplicationRecord
  has_many :poll_options, dependent: :destroy
  has_many :poll_votes, dependent: :destroy
  has_many :voters, through: :poll_votes, source: :member

  validates :question, presence: true
  validates :status, inclusion: { in: %w[open closed expired] }
  validates :allow_multiple, inclusion: { in: [true, false] }

  scope :open, -> { where(status: "open") }
  scope :closed, -> { where(status: "closed") }
  scope :recent, -> { order(created_at: :desc) }
end
