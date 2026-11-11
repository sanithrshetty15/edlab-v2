class Semester < ApplicationRecord
  validates :name, presence: true
  validates :start_date, presence: true
  validates :end_date, presence: true

  scope :active, -> { where("end_date >= ?", Date.current) }
  scope :ordered, -> { order(start_date: :desc) }

  def self.current
    find_by(is_current: true) || where("start_date <= ? AND end_date >= ?", Date.current, Date.current).order(start_date: :desc).first
  end
end
