class DeviceRegistration < ApplicationRecord
  has_many :biometric_enrollments, dependent: :destroy
  has_many :device_events, dependent: :destroy

  validates :name, presence: true
  validates :device_type, presence: true
  validates :api_key_digest, presence: true, uniqueness: true

  scope :active, -> { where(status: "active") }
  scope :by_device_type, ->(type) { where(device_type: type) }
end
