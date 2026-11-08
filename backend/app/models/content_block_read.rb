class ContentBlockRead < ApplicationRecord
  belongs_to :content_block
  belongs_to :member

  validates :member_id, uniqueness: { scope: :content_block_id, message: "already read this content block" }

  scope :for_block, ->(block_id) { where(content_block_id: block_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
end
