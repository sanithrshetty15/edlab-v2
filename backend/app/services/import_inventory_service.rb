class ImportInventoryService
  class ImportError < StandardError; end

  REQUIRED_COLUMNS = %w[name category].freeze
  OPTIONAL_COLUMNS = %w[description total_quantity storage_location condition].freeze

  def self.import(file:, imported_by:)
    filename = file.original_filename
    extension = File.extname(filename).downcase

    rows = parse_file(file, extension)
    validate_columns(rows.first)

    ActiveRecord::Base.transaction do
      import_record = InventoryImport.create!(
        filename: filename,
        total_rows: rows.size,
        imported_by: imported_by
      )

      errors = []

      rows.each_with_index do |row, index|
        next if row.all? { |_, v| v.blank? }

        begin
          process_row(row, import_record, index + 2)
        rescue => e
          errors << { row: index + 2, error: e.message }
          import_record.update!(error_rows: import_record.error_rows + 1)
        end
      end

      import_record.update!(
        status: errors.empty? ? "completed" : "completed_with_errors",
        imported_rows: import_record.total_rows - import_record.error_rows
      )

      {
        success: true,
        import_record: import_record,
        errors: errors
      }
    end
  rescue => e
    Rails.logger.error("Inventory import failed: #{e.class} - #{e.message}")
    {
      success: false,
      error: "Import failed: #{e.message}",
      status: :unprocessable_entity
    }
  end

  def self.parse_file(file, extension)
    case extension
    when ".csv"
      parse_csv(file)
    when ".xlsx", ".xls"
      parse_excel(file)
    else
      raise "Unsupported file format: #{extension}"
    end
  end

  def self.parse_csv(file)
    CSV.parse(file.read, headers: true).map(&:to_hash)
  rescue => e
    raise "Failed to parse CSV: #{e.message}"
  end

  def self.parse_excel(file)
    require "roo" unless defined?(Roo)
    spreadsheet = Roo::Spreadsheet.open(file.path)
    spreadsheet.sheet(0).parse(headers: true).map(&:to_hash)
  rescue LoadError
    raise "Roo gem is required for Excel imports. Add 'roo' to Gemfile."
  rescue => e
    raise "Failed to parse Excel: #{e.message}"
  end

  def self.validate_columns(header_row)
    column_names = header_row.keys.map(&:to_s).map { |c| c.downcase.strip.gsub(/\s/, "_") }

    missing = REQUIRED_COLUMNS - column_names
    unless missing.empty?
      raise "Missing required columns: #{missing.join(', ')}"
    end
  end

  def self.process_row(row, import_record, row_number)
    name = row["name"].to_s.strip
    category = row["category"].to_s.strip
    total_quantity = row["total_quantity"].to_s.strip.to_i
    description = row["description"].to_s.strip.presence
    storage_location = row["storage_location"].to_s.strip.presence
    condition = row["condition"].to_s.strip.presence

    if name.blank? || category.blank?
      raise "Name and category are required"
    end

    if total_quantity.negative?
      raise "Total quantity cannot be negative"
    end

    existing = InventoryItem
      .where(is_active: true, deleted_at: nil)
      .find_by(name: name, category: category)

    if existing
      existing.update!(
        total_quantity: [existing.total_quantity + total_quantity, 0].max,
        available_quantity: [existing.available_quantity + total_quantity, existing.total_quantity + total_quantity].min,
        description: description.presence || existing.description,
        storage_location: storage_location.presence || existing.storage_location,
        condition: condition.presence || existing.condition
      )
    else
      InventoryItem.create!(
        name: name,
        category: category,
        description: description,
        total_quantity: total_quantity,
        available_quantity: total_quantity,
        storage_location: storage_location,
        condition: condition
      )
    end
  end
end
