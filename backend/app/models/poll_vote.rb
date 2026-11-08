class PollVote < ApplicationRecord
  belongs_to :poll
  belongs_to :poll_option
  belongs_to :member

  validates :member_id, uniqueness: { scope: :poll_id, message: "already voted on this poll" }

  scope :for_poll, ->(poll_id) { where(poll_id: poll_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
end
