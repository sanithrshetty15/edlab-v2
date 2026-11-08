class LabRulesAcceptance < ApplicationRecord
  belongs_to :member
  belongs_to :lab_rules_version

  validates :member_id, uniqueness: { scope: :lab_rules_version_id, message: "already accepted this version" }

  scope :for_version, ->(version_id) { where(lab_rules_version_id: version_id) }
  scope :after_date, ->(date) { where(created_at: date..) }
end
