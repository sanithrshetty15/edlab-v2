class Faq < ApplicationRecord
  belongs_to :category, class_name: "FaqCategorie", foreign_key: "category_id", optional: true
  has_many :faq_items, class_name: "Faq", foreign_key: "parent_id", dependent: :destroy

  validates :slug, presence: true, uniqueness: true
  validates :short_answer_length, numericality: { greater_than_or_equal_to: 1 }, allow_nil: true

  scope :with_category, -> { joins(:category) }
  scope :search, ->(query) { where("title ILIKE ? OR question ILIKE ?", "%#{query}%", "%#{query}%") }
end
