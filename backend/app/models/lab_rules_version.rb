class LabRulesVersion < ApplicationRecord
  has_many :lab_rules_acceptances, dependent: :nullify

  validates :version_number, presence: true, uniqueness: true
  validates :content, presence: true

  scope :ordered, -> { order(version_number: :desc) }

  def self.current
    find_by(is_current: true) || order(created_at: :desc).first
  end
end
