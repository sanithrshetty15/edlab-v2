class AcademicYear < ApplicationRecord
  validates :name, presence: true, uniqueness: true
  validates :start_year, presence: true, numericality: { only_integer: true }
  validates :end_year, presence: true, numericality: { only_integer: true }

  scope :active, -> { where("end_year >= ?", Date.current.year) }
  scope :ordered, -> { order(start_year: :desc) }

  def self.current
    where("start_year <= ? AND end_year >= ?", Date.current.year, Date.current.year).first
  end
end
