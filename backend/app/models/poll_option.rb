class PollOption < ApplicationRecord
  belongs_to :poll
  has_many :poll_votes, dependent: :destroy

  validates :label, presence: true, length: { maximum: 255 }

  scope :for_poll, ->(poll_id) { where(poll_id: poll_id) }
  scope :ordered, -> { order(position: :asc) }
end
