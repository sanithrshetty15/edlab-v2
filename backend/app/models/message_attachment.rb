class MessageAttachment < ApplicationRecord
  belongs_to :message
  has_one_attached :file

  validates :name, presence: true, length: { maximum: 255 }
  validates :file_type, inclusion: { in: %w[image document pdf other] }
  validates :file_size, numericality: { greater_than: 0 }

  scope :for_message, ->(message_id) { where(message_id: message_id) }
  scope :images, -> { where(file_type: "image") }
  scope :documents, -> { where(file_type: "document") }
end
