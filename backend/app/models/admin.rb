class Admin < ApplicationRecord
  belongs_to :user

  validates :name, presence: true

  validates :user_id, uniqueness: true

  scope :primary, -> { where(is_primary: true) }
  scope :secondary, -> { where(is_primary: false) }
end
