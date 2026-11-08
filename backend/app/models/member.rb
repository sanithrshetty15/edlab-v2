class Member < ApplicationRecord
  include PhoneValidation
  include LinkUrlValidator

  belongs_to :user
  has_many :lab_rules_acceptances, dependent: :destroy
  has_many :biometric_enrollments, dependent: :destroy
  has_many :attendance_logs, dependent: :destroy
  has_many :borrow_records, dependent: :destroy
  has_many :return_requests, dependent: :destroy
  has_many :task_submission_statuses, dependent: :destroy
  has_many :contact_messages, dependent: :destroy

  validates :name, presence: true, length: { maximum: 255 }
  validates :login_id, presence: true, uniqueness: true, format: { with: /\A[\w.-]+\z/, message: "can only contain letters, numbers, dots, and hyphens" }
  validates :academic_year, presence: true, numericality: { only_integer: true, greater_than_or_equal_to: 1, less_than_or_equal_to: 8 }
  validates :branch, presence: true, length: { maximum: 100 }
  validates :email, presence: true, uniqueness: true, format: { with: URI::MailTo::EMAIL_REGEXP }
  validates :phone, presence: true
  validates :usn, uniqueness: true, allow_nil: true, allow_blank: true, format: { with: /\A[A-Z]{2,3}\d{2}[A-Z]{2}\d{3,4}\z/, message: "must be in format like 22H51A05P6", allow_blank: true }
  validates :roll_number, uniqueness: true, allow_nil: true, allow_blank: true, format: { with: /\A[\w-]+\z/, message: "contains invalid characters", allow_blank: true }
  validates :status, inclusion: { in: %w[active inactive deactivated] }
  %w[github_url linkedin_url portfolio_url].each do |field|
    validates field, format: { with: URI::DEFAULT_PARSER.make_regexp, message: "must be a valid URL", allow_blank: true }
  end

  validates :phone, format: { with: /\A\+?[1-9]\d{7,14}\z/, message: "must be a valid phone number" }, allow_blank: true

  scope :active, -> { where(status: "active", deleted_at: nil) }
  scope :by_academic_year, ->(year) { where(academic_year: year) }
  scope :by_branch, ->(branch) { where(branch: branch) }
  scope :search, ->(query) { where("name ILIKE ? OR login_id ILIKE ? OR email ILIKE ?", "%#{query}%", "%#{query}%", "%#{query}%") }

  def full_profile?
    usn.present? || roll_number.present?
  end
end
