class AddPhoneVerifiedToMembers < ActiveRecord::Migration[8.1]
  def change
    add_column :members, :phone_verified, :boolean, default: false, null: false
  end
end
