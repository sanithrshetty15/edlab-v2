module Api
  module V1
    class HealthController < BaseController
      def show
        render json: {
          status: "ok",
          application: "Edwins Lab API",
          version: "v1"
        }, status: :ok
      end
    end
  end
end
