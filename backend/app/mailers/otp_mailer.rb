class OtpMailer < ApplicationMailer
  def verification_email(user, code)
    @user = user
    @code = code

    attachments.inline["edwins.png"] = File.read(
      Rails.root.join("app/assets/images/edwins.png")
    )

    mail(
      to: @user.email,
      subject: "Your Edwins Lab Verification Code"
    )
  end
end