class LabRulesVersion < ApplicationRecord
  has_many :lab_rules_acceptances, dependent: :nullify

  validates :version_number, presence: true
  validates :content, presence: true
  validates :effective_from, presence: true

  scope :current, -> { where(effective_from: ..Time.current).order(effective_from: :desc).first }
  scope :active, -> { where("effective_from <= ? AND (expires_at IS NULL OR expires_at >= ?)", Time.current, Time.current) }
end
