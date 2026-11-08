class FaqCategorie < ApplicationRecord
  has_many :faqs, foreign_key: "category_id", dependent: :restrict_with_exception
  has_many :children, class_name: "FaqCategorie", foreign_key: "parent_id", dependent: :restrict_with_exception
  belongs_to :parent, class_name: "FaqCategorie", foreign_key: "parent_id", optional: true

  validates :name, presence: true, length: { maximum: 255 }
  validates :slug, presence: true, uniqueness: true
  validates :description, length: { maximum: 500 }
  validates :position, numericality: { greater_than_or_equal_to: 0 }, allow_nil: true

  scope :active, -> { where(active: true) }
  scope :ordered, -> { order(position: :asc, name: :asc) }
end
