class ApplicationMailer < ActionMailer::Base
  default from: "Edwins Lab <#{ENV.fetch('GMAIL_USERNAME')}>"
  layout "mailer"
end