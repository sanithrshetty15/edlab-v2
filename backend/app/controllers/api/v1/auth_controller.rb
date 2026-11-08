module Api
  module V1
    class AuthController < BaseController
      before_action :authenticate_user!, only: [:logout, :me, :change_password]
      def member_login
        login_type = params[:login_type]
        login_id = params[:login_id]&.strip
        password = params[:password]

        unless %w[usn roll_number].include?(login_type)
          return render json: {
            success: false,
            message: "Invalid login type"
          }, status: :unprocessable_entity
        end

        if login_id.blank? || password.blank?
          return render json: {
            success: false,
            message: "Login ID and password are required"
          }, status: :unprocessable_entity
        end

        column = login_type == "usn" ? :usn : :roll_number

        member = Member.find_by(column => login_id)

        unless member &&
               member.status == "active" &&
               member.deleted_at.nil? &&
               member.user.active? &&
               member.user.authenticate(password)
          return render json: {
            success: false,
            message: "Invalid login credentials"
          }, status: :unauthorized
        end

        session, token = Session.create_for(
          member.user,
          ip_address: request.remote_ip,
          user_agent: request.user_agent
        )

        render json: {
          success: true,
          message: "Login successful",
          token: token,
          must_change_password: member.must_change_password,
          user: {
            id: member.user.id,
            name: member.name,
            email: member.email,
            role: member.user.role,
            login_type: login_type
          }
        }, status: :ok
      end

      def admin_login
  email = params[:email]&.strip&.downcase
  password = params[:password]

  if email.blank? || password.blank?
    return render json: {
      success: false,
      message: "Email and password are required"
    }, status: :unprocessable_entity
  end

  user = User.find_by(email: email)

  unless user &&
         user.admin? &&
         user.active? &&
         user.authenticate(password)
    return render json: {
      success: false,
      message: "Invalid login credentials"
    }, status: :unauthorized
  end

  session, token = Session.create_for(
    user,
    ip_address: request.remote_ip,
    user_agent: request.user_agent
  )

  render json: {
    success: true,
    message: "Admin login successful",
    token: token,
    admin: {
      id: user.admin.id,
      name: user.admin.name,
      email: user.email,
      is_primary: user.admin.is_primary
    }
  }, status: :ok
end

      def logout
  current_session.destroy!

  render json: {
    success: true,
    message: "Logged out successfully"
  }, status: :ok
end

def me
  user_data = {
    id: current_user.id,
    email: current_user.email,
    role: current_user.role
  }

  if current_user.admin?
    user_data[:admin] = {
      id: current_user.admin.id,
      name: current_user.admin.name,
      is_primary: current_user.admin.is_primary
    }
  end

  member = current_user.member

  if member
    user_data[:member] = {
      id: member.id,
      name: member.name,
      usn: member.usn,
      roll_number: member.roll_number,
      must_change_password: member.must_change_password
    }
  end

  render json: {
    success: true,
    user: user_data
  }, status: :ok
end

def change_password
  current_password = params[:current_password]
  new_password = params[:new_password]
  password_confirmation = params[:password_confirmation]

  if current_password.blank? || new_password.blank? || password_confirmation.blank?
    return render json: {
      success: false,
      message: "All password fields are required"
    }, status: :unprocessable_entity
  end

  unless current_user.authenticate(current_password)
    return render json: {
      success: false,
      message: "Current password is incorrect"
    }, status: :unauthorized
  end

  if new_password != password_confirmation
    return render json: {
      success: false,
      message: "New password and confirmation do not match"
    }, status: :unprocessable_entity
  end

  if new_password.length < 8
    return render json: {
      success: false,
      message: "Password must contain at least 8 characters"
    }, status: :unprocessable_entity
  end

  ActiveRecord::Base.transaction do
  current_user.update!(
    password: new_password,
    password_confirmation: password_confirmation
  )

  member = current_user.member
  member&.update!(must_change_password: false)
  current_user.sessions.where.not(id: current_session.id).destroy_all
end

  render json: {
    success: true,
    message: "Password changed successfully",
    must_change_password: false
  }, status: :ok
end
    end
  end
end