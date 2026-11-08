module LinkUrlValidator
  extend ActiveSupport::Concern

  included do
    %w[github_url linkedin_url portfolio_url].each do |field|
      validates field, format: { with: URI::DEFAULT_PARSER.make_regexp, message: "must be a valid URL", allow_blank: true }
    end
  end
end
