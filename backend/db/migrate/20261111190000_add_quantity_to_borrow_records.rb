class AddQuantityToBorrowRecords < ActiveRecord::Migration[8.1]
  def change
    add_column :borrow_records, :quantity, :integer, default: 1, null: false
  end
end
