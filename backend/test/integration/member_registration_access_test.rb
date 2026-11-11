require "test_helper"

class MemberRegistrationAccessTest < ActionDispatch::IntegrationTest
  REGISTRATION_PAYLOAD = {
    name: "Public Registration Test",
    login_id: "public.registration.test",
    email: "public.registration.test@edwinslab.local",
    phone: "+919876543210",
    usn: "4AL25CS999",
    roll_number: "PUBLIC-REG-TEST",
    academic_year: 2,
    branch: "Computer Science",
    section: "A",
    password: "PublicPass#2026",
    password_confirmation: "PublicPass#2026"
  }.freeze

  test "public member registration is rejected" do
    assert_no_difference(["User.count", "Member.count"]) do
      post "/api/v1/members/register", params: REGISTRATION_PAYLOAD, as: :json
    end

    assert_response :forbidden
    body = JSON.parse(response.body)
    assert_equal false, body["success"]
    assert_match(/disabled/i, body["message"])
  end

  test "member management creation requires an authenticated admin" do
    post "/api/v1/members-management", params: REGISTRATION_PAYLOAD, as: :json

    assert_response :unauthorized
  end

  test "an authenticated member cannot create another member" do
    member_user = User.create!(
      email: "registration-access-member@edwinslab.local",
      password: "MemberPass#2026",
      role: "member",
      active: true
    )
    _session, token = Session.create_for(member_user)

    post "/api/v1/members-management",
      params: REGISTRATION_PAYLOAD,
      headers: { "Authorization" => "Bearer #{token}" },
      as: :json

    assert_response :forbidden
  end

  test "an authenticated admin can create a member" do
    admin_user = User.create!(
      email: "registration-access-admin@edwinslab.local",
      password: "AdminPass#2026",
      role: "admin",
      active: true
    )
    _session, token = Session.create_for(admin_user)

    assert_difference(["User.count", "Member.count"], 1) do
      post "/api/v1/members-management",
        params: REGISTRATION_PAYLOAD,
        headers: { "Authorization" => "Bearer #{token}" },
        as: :json
    end

    assert_response :created
    body = JSON.parse(response.body)
    assert_equal true, body["success"]
    assert_equal REGISTRATION_PAYLOAD[:login_id], body.dig("member", "login_id")
    assert Member.exists?(login_id: REGISTRATION_PAYLOAD[:login_id])
  end
end
