module Api
  module V1
    class BaseController < ActionController::API
      PaginationResult = Struct.new(:items, :current_page, :per_page, :total_count, keyword_init: true) do
        def total_pages
          return 0 if per_page.to_i.zero?
          (total_count.to_f / per_page).ceil
        end
      end

      rescue_from ActiveRecord::RecordNotFound do |error|
        render json: { success: false, message: "Record not found" }, status: :not_found
      end

      rescue_from ActiveRecord::RecordInvalid do |error|
        render json: { success: false, message: error.record.errors.full_messages.to_sentence, errors: error.record.errors.full_messages }, status: :unprocessable_entity
      end

      private

      def paginate(scope, default_per_page: 20)
        page = [params[:page].to_i, 1].max
        per_page = params[:per_page].to_i
        per_page = default_per_page if per_page <= 0
        per_page = [[per_page, 100].min, 1].max

        PaginationResult.new(
          items: scope.offset((page - 1) * per_page).limit(per_page),
          current_page: page,
          per_page: per_page,
          total_count: scope.unscope(:order, :select).count
        )
      end

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