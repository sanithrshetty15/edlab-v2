module Api
  module V1
    class BaseController < ActionController::API
      private

      def authenticate_user!
        authorization = request.headers["Authorization"]
token = authorization&.match(/\ABearer\s+(\S+)\z/i)&.captures&.first

        if token.blank?
          return render json: {
            success: false,
            message: "Authentication token is required"
          }, status: :unauthorized
        end

        @current_session = Session.find_by(
          token_digest: Session.digest_token(token)
        )

        unless @current_session && !@current_session.expired?
          return render json: {
            success: false,
            message: "Invalid or expired session"
          }, status: :unauthorized
        end

        @current_user = @current_session.user

        unless @current_user.active?
          return render json: {
            success: false,
            message: "Account is inactive"
          }, status: :forbidden
        end

        @current_session.update!(
  last_active_at: Time.current,
  expires_at: @current_user.admin? ? 60.minutes.from_now : @current_session.expires_at
)
      end

      def require_password_change_completed!
  return if current_user.admin?

  member = current_user.member

  if member&.must_change_password?
    return render json: {
      success: false,
      message: "You must change your password before accessing this feature",
      must_change_password: true
    }, status: :forbidden
  end
end
      attr_reader :current_user, :current_session
    end
  end
end