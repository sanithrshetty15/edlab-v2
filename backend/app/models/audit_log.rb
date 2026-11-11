class AuditLog < ApplicationRecord
  belongs_to :user, optional: true

  validates :action, presence: true
  validates :resource_type, presence: true

  scope :for_resource, ->(resource_type, resource_id) { where(resource_type: resource_type, resource_id: resource_id) }
  scope :for_user, ->(user_id) { where(user_id: user_id) }
  scope :by_action, ->(action) { where(action: action) }
  scope :by_resource_type, ->(type) { where(resource_type: type) }
  scope :recent, -> { order(created_at: :desc) }
  scope :for_today, -> { where(created_at: Time.current.beginning_of_day..Time.current.end_of_day) }

  def self.log(action:, user: nil, resource: nil, resource_type: nil, resource_id: nil, details: {}, metadata: {}, ip_address: nil)
    resolved_type = resource_type || resource&.class&.name
    resolved_id = resource_id || (resource&.respond_to?(:id) ? resource.id : nil)
    create!(
      user: user,
      action: action.to_s,
      resource_type: resolved_type.to_s,
      resource_id: resolved_id,
      details: (details.presence || metadata).with_indifferent_access,
      ip_address: ip_address
    )
  end
end
