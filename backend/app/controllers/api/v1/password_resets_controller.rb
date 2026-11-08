module Api
  module V1
    class PasswordResetsController < BaseController
      def create
        login_id = params[:login_id].to_s.strip
        delivery_method = params[:delivery_method].to_s

        unless login_id.present? && delivery_method == "email"
          return render json: {
            success: false,
            message: "A login ID and email delivery method are required"
          }, status: :unprocessable_entity
        end

        member = Member.active.find_by(login_id: login_id)

        if member && member.user.active?
          OtpDeliveryService.send_email(
            user: member.user,
            purpose: "password_reset"
          )
        end

        render json: {
          success: true,
          message: "If the account exists, a verification code will be sent"
        }, status: :ok
      end

      def update
        login_id = params[:login_id].to_s.strip
        code = params[:otp].to_s
        new_password = params[:new_password].to_s
        password_confirmation = params[:password_confirmation].to_s

        member = Member.active.find_by(login_id: login_id)

        unless member && member.user.active?
          return password_reset_error("Invalid login ID or verification code")
        end

        unless new_password.length >= 8
          return password_reset_error(
            "Password must contain at least 8 characters"
          )
        end

        unless new_password == password_confirmation
          return password_reset_error(
            "Password confirmation does not match"
          )
        end

        member.user.with_lock do
          otp = member.user.otp_codes
            .where(
              purpose: "password_reset",
              delivery_method: "email"
            )
            .unused
            .unexpired
            .order(created_at: :desc)
            .first

          unless otp&.verify_code(code)
            return password_reset_error(
              "Invalid or expired verification code"
            )
          end

          member.user.update!(password: new_password)
          member.update!(must_change_password: false)
          member.user.sessions.destroy_all
        end

        render json: {
          success: true,
          message: "Password reset successfully. Please log in again."
        }, status: :ok

      rescue ActiveRecord::RecordInvalid
        password_reset_error("Unable to reset password")
      end

      private

      def password_reset_error(message)
        render json: {
          success: false,
          message: message
        }, status: :unprocessable_entity
      end
    end
  end
end