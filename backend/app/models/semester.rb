class Semester < ApplicationRecord
  has_many :periods, class_name: "Semester", foreign_key: "parent_id", dependent: :nullify
  belongs_to :parent, class_name: "Semester", optional: true

  validates :name, presence: true
  validates :code, presence: true, uniqueness: true
  validates :start_date, presence: true
  validates :end_date, presence: true

  scope :current, -> { where("start_date <= ? AND end_date >= ?", Time.current, Time.current).first }
  scope :active, -> { where("end_date >= ?", Time.current) }
  scope :ordered, -> { order(start_date: :desc) }
end
