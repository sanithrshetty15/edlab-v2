class AuditLog < ApplicationRecord
  belongs_to :actor, polymorphic: true, optional: true
  belongs_to :resource, polymorphic: true, optional: true

  validates :action, presence: true

  scope :for_resource, ->(resource_type, resource_id) { where(resource_type: resource_type, resource_id: resource_id) }
  scope :for_actor, ->(actor) { where(actor: actor) }
  scope :by_action, ->(action) { where(action: action) }
  scope :by_resource_type, ->(type) { where(resource_type: type) }
  scope :recent, -> { order(created_at: :desc) }
  scope :for_today, -> { where(created_at: Time.current.beginning_of_day..Time.current.end_of_day) }

  def self.log(action:, actor: nil, resource: nil, metadata: {}, ip_address: nil)
    create!(
      action: action,
      actor: actor,
      resource: resource,
      metadata: metadata,
      ip_address: ip_address,
      user_agent: nil # Will be set by controller concern
    )
  end
end
