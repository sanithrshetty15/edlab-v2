class GroupSetting < ApplicationRecord
  validates :key, presence: true, uniqueness: true

  scope :for_key, ->(key) { where(key: key) }

  def self.get(key, default = nil)
    find_by(key: key)&.value || default
  end

  def self.set(key, value)
    find_or_create_by(key: key).update(value: value)
  end
end
