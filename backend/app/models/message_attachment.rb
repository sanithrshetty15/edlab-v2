class MessageAttachment < ApplicationRecord
  belongs_to :group_message

  validates :filename, presence: true
  validates :content_type, presence: true
  validates :byte_size, numericality: { greater_than: 0 }
  validates :storage_key, presence: true
end
