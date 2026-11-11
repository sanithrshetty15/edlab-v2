require "test_helper"

class BorrowingExtensionTest < ActionDispatch::IntegrationTest
  setup do
    @user = User.create!(
      email: "borrowing-extension-member@edwinslab.local",
      password: "MemberPass#2026",
      role: "member",
      active: true
    )
    @member = Member.create!(
      user: @user,
      name: "Borrowing Extension Member",
      login_id: "borrowing.extension.member",
      email: "borrowing.extension.member@edwinslab.local",
      phone: "+919876543211",
      academic_year: 2,
      branch: "Computer Science",
      status: "active",
      must_change_password: false
    )
    @item = InventoryItem.create!(
      name: "Extension Test Item",
      category: "Equipment",
      total_quantity: 2,
      available_quantity: 1
    )
    @borrow_record = BorrowRecord.create!(
      member: @member,
      inventory_item: @item,
      quantity: 1,
      status: "approved",
      borrowed_at: Time.current,
      due_date: 1.day.ago.to_date
    )
    _session, @token = Session.create_for(@user)
  end

  test "requests an extension for an overdue borrow record" do
    assert_routing(
      { method: "post", path: "/api/v1/borrowing/extension-request" },
      { controller: "api/v1/borrowing", action: "request_extension" }
    )

    requested_return_date = Date.current + 7.days

    assert_difference("ExtensionRequest.count", 1) do
      post "/api/v1/borrowing/extension-request",
        params: {
          borrow_record_id: @borrow_record.id,
          reason: "Need more time to finish the project",
          requested_return_date: requested_return_date.to_s
        },
        headers: { "Authorization" => "Bearer #{@token}" },
        as: :json
    end

    assert_response :created
    body = JSON.parse(response.body)
    assert_equal true, body["success"]
    assert_equal @borrow_record.id, body.dig("extension", "borrow_record_id")
    assert_equal "pending", body.dig("extension", "status")
    assert_equal requested_return_date.to_s, body.dig("extension", "requested_return_date")

    get "/api/v1/borrowing/requests/#{@borrow_record.id}",
      headers: { "Authorization" => "Bearer #{@token}" },
      as: :json

    assert_response :ok
    show_body = JSON.parse(response.body)
    assert_equal "pending", show_body.dig("request", "extension_request", "status")
  end

  test "rejects an extension request for a borrow record that is not overdue" do
    @borrow_record.update!(due_date: Date.current + 2.days)

    assert_no_difference("ExtensionRequest.count") do
      post "/api/v1/borrowing/extension-request",
        params: {
          borrow_record_id: @borrow_record.id,
          reason: "Need more time"
        },
        headers: { "Authorization" => "Bearer #{@token}" },
        as: :json
    end

    assert_response :unprocessable_entity
    body = JSON.parse(response.body)
    assert_equal false, body["success"]
    assert_match(/overdue/i, body["message"])
  end
end
