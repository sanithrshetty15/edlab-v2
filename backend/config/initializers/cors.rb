Rails.application.config.middleware.insert_before 0, Rack::Cors do
  allow do
    allowed_origins = ENV.fetch("CORS_ORIGINS", "http://localhost:5173,http://localhost:3000")
      .split(",")
      .map(&:strip)
      .reject(&:empty?)

    allowed_origins.concat([
      "http://localhost",
      "https://localhost",
      "capacitor://localhost",
      "https://edwinslab.vercel.app"
    ])

    origins(*allowed_origins.uniq)

    resource "*",
      headers: :any,
      methods: %i[get post put patch delete options head]
  end
end
