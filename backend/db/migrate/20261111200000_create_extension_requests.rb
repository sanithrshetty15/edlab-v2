class CreateExtensionRequests < ActiveRecord::Migration[8.1]
  def change
    create_table :extension_requests do |t|
      t.references :borrow_record, null: false, foreign_key: true, index: { unique: true }
      t.references :member, null: false, foreign_key: true
      t.date :requested_return_date, null: false
      t.text :reason, null: false
      t.string :status, null: false, default: "pending"

      t.timestamps
    end
  end
end
