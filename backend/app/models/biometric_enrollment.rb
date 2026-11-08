class BiometricEnrollment < ApplicationRecord
  belongs_to :member
  belongs_to :device_registration

  validates :member_id, uniqueness: { scope: :device_registration_id, message: "already has a fingerprint enrolled on this device" }

  scope :for_device, ->(device_id) { where(device_registration_id: device_id) }
  scope :for_member, ->(member_id) { where(member_id: member_id) }
end
