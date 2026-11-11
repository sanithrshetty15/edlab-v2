class GroupSetting < ApplicationRecord
  validates :permission, inclusion: { in: %w[everyone admins_only] }

  def self.get(key, default = nil)
    setting = first
    return default unless setting
    setting.respond_to?(key) ? setting.public_send(key) : default
  end
end
