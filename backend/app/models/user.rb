class User < ApplicationRecord
  has_secure_password

  has_one :member
  has_one :admin
  has_many :sessions, dependent: :destroy
  has_many :otp_codes, dependent: :destroy
  has_many :notifications, dependent: :destroy

  validates :email, presence: true, uniqueness: true
  validates :role, presence: true

  scope :active, -> { where(active: true) }

  def admin?
    admin.present? || %w[admin primary_admin].include?(role)
  end

  def member?
    member.present? || role == "member"
  end

  def primary_admin?
    admin&.is_primary == true || role == "primary_admin"
  end
end
