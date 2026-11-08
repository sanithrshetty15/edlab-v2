class User < ApplicationRecord
  has_secure_password

  has_one :member
  has_one :admin
  has_many :sessions, dependent: :destroy
  has_many :otp_codes, dependent: :destroy
  
  validates :email, presence: true, uniqueness: true
  validates :role, presence: true

  def admin?
    admin.present?
  end

    def primary_admin?
    admin&.is_primary == true
  end
end
