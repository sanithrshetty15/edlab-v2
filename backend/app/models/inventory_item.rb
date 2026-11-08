class InventoryItem < ApplicationRecord
  has_many :borrow_records, dependent: :restrict_with_exception

  validates :name, presence: true, length: { maximum: 255 }
  validates :category, presence: true
  validates :quantity, presence: true, numericality: { greater_than_or_equal_to: 0, only_integer: true }
  validates :uom, presence: true, length: { maximum: 50 }
  validates :location, length: { maximum: 255 }
  validates :hsn_code, length: { maximum: 100 }
  validates :brand, length: { maximum: 100 }
  validates :model, length: { maximum: 100 }
  validate :warranty_upto_is_valid_date
  validates :state, inclusion: { in: %w[new like_new good fair damaged lost] }, allow_nil: true
  validates :slug, uniqueness: true, allow_nil: true
  validates :min_qty_for_warn, numericality: { greater_than_or_equal_to: 0 }, allow_nil: true

  before_validation :generate_slug, on: :create
  before_update :recalculate_total_value

  scope :by_category, ->(category) { where(category: category) }
  scope :by_location, ->(location) { where(location: location) }
  scope :low_stock, ->(min_qty = 5) { where("quantity <= ?", min_qty) }
  scope :active, -> { where(active: true) }
  scope :search, ->(query) { where("name ILIKE ? OR description ILIKE ? OR category ILIKE ?", "%#{query}%", "%#{query}%", "%#{query}%") }

  # Returns combined brand and model string
  def brand_and_model
    "#{brand} #{model}".strip.presence
  end

  # Returns true if brand/model is not blank
  def brand_and_model?
    !brand_and_model.blank?
  end

  private

  def warranty_upto_is_valid_date
    return if warranty_upto.blank?
    begin
      Date.parse(warranty_upto.to_s) rescue errors.add(:warranty_upto, "must be a valid date")
    rescue
      # already handled
    end
  end

  def generate_slug
    self.slug = name.parameterize if slug.blank?
  end

  def recalculate_total_value
    self.total_value = if unit_cost.present? && quantity.present?
      unit_cost * quantity
    else
      0
    end
  end
end
