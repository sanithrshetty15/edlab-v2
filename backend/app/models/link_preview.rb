class LinkPreview < ApplicationRecord
  belongs_to :group_message, optional: true

  validates :url, presence: true
end
