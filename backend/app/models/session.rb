require "securerandom"
require "digest"

class Session < ApplicationRecord
  belongs_to :user

  validates :token_digest, presence: true, uniqueness: true
  validates :last_active_at, presence: true

  scope :active, -> {
    where("expires_at IS NULL OR expires_at > ?", Time.current)
  }

  def expired?
    expires_at.present? && expires_at <= Time.current
  end

  def self.generate_token
    SecureRandom.hex(32)
  end

  def self.digest_token(token)
    Digest::SHA256.hexdigest(token)
  end

  def self.create_for(user, ip_address: nil, user_agent: nil)
    token = generate_token

    session = user.sessions.create!(
      token_digest: digest_token(token),
      ip_address: ip_address,
      user_agent: user_agent,
      last_active_at: Time.current,
      expires_at: user.admin? ? 60.minutes.from_now : 7.days.from_now
    )
    [session, token]
  end
end
