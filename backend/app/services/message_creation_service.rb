class MessageCreationService
  MAX_CONTENT_LENGTH = 10_000
  MAX_ATTACHMENTS = 5
  MAX_ATTACHMENT_SIZE = 10.megabytes

  ALLOWED_CONTENT_TYPES = [
    "image/jpeg",
    "image/png",
    "image/gif",
    "image/webp",
    "application/pdf",
    "text/plain",
    "application/zip"
  ]

  class MessageError < StandardError; end

  def self.create(params:, user:, ip_address: nil)
    content = params[:content].to_s.strip
    attachments = params[:attachments] || []

    validate_message_content(content, attachments)

    ActiveRecord::Base.transaction do
      message = GroupMessage.create!(
        user_id: user.id,
        content: content,
        message_type: determine_message_type(content, attachments),
        is_edited: false,
        is_deleted: false,
        is_pinned: false
      )

      attachments.each do |attachment_data|
        MessageAttachment.create!(
          group_message_id: message.id,
          filename: attachment_data[:filename],
          content_type: attachment_data[:content_type],
          byte_size: attachment_data[:byte_size],
          storage_key: attachment_data[:storage_key]
        )
      end

      create_receipts_for_message(message, user)

      if content.match?(/\Ahttps?:\/\//i)
        fetch_and_create_link_preview(message, content)
      end

      message
    end

    {
      success: true,
      message: message
    }
  rescue MessageError => e
    {
      success: false,
      error: e.message,
      status: :unprocessable_entity
    }
  rescue => e
    Rails.logger.error("Message creation failed: #{e.class} - #{e.message}")
    {
      success: false,
      error: "Failed to create message",
      status: :internal_server_error
    }
  end

  def self.validate_message_content(content, attachments)
    raise MessageError, "Message content is required" if content.blank? && attachments.blank?

    if content.length > MAX_CONTENT_LENGTH
      raise MessageError, "Message content is too long (max #{MAX_CONTENT_LENGTH} characters)"
    end

    if attachments.is_a?(Array) && attachments.size > MAX_ATTACHMENTS
      raise MessageError, "Maximum #{MAX_ATTACHMENTS} attachments allowed"
    end

    if attachments.is_a?(Array)
      attachments.each do |attachment|
        if attachment[:byte_size].to_i > MAX_ATTACHMENT_SIZE
          raise MessageError, "Attachment too large (max #{MAX_ATTACHMENT_SIZE / 1.megabyte}MB)"
        end

        unless ALLOWED_CONTENT_TYPES.include?(attachment[:content_type])
          raise MessageError, "File type not allowed: #{attachment[:content_type]}"
        end
      end
    end
  end

  def self.determine_message_type(content, attachments)
    if attachments.is_a?(Array) && attachments.any?
      "media"
    elsif content.match?(/\Ahttps?:\/\//i)
      "link"
    else
      "text"
    end
  end

  def self.create_receipts_for_message(message, sender)
    all_user_ids = User
      .joins(:member)
      .where(members: { deleted_at: nil, status: "active" })
      .or(User.where(role: "admin", active: true))
      .pluck(:id)

    all_user_ids.each do |user_id|
      MessageReceipt.create!(
        group_message_id: message.id,
        user_id: user_id,
        delivered_at: Time.current
      )
    end
  end

  def self.fetch_and_create_link_preview(message, url)
    return unless message.content.match?(/\Ahttps?:\/\/\S+\z/)

    begin
      uri = URI(url)
      Net::HTTP.start(uri.hostname, uri.port, use_ssl: uri.scheme == "https") do |http|
        request = Net::HTTP::Get.new(uri)
        response = http.request(request)

        if response.is_a?(Net::HTTPSuccess)
          html = response.body

          title_match = html.match(/<title[^>]*>([^<]*)<\/title>/i)
          description_match = html.match(/<meta[^>]*name=["']description["'][^>]*content=["']([^"']*)["']/i)
          og_title_match = html.match(/<meta[^>]*property=["']og:title["'][^>]*content=["']([^"']*)["']/i)
          og_desc_match = html.match(/<meta[^>]*property=["']og:description["'][^>]*content=["']([^"']*)["']/i)
          og_image_match = html.match(/<meta[^>]*property=["']og:image["'][^>]*content=["']([^"']*)["']/i)
          og_site_match = html.match(/<meta[^>]*property=["']og:site_name["'][^>]*content=["']([^"']*)["']/i)

          LinkPreview.create!(
            group_message_id: message.id,
            url: url,
            title: (og_title_match || title_match)&.captures&.first,
            description: (og_desc_match || description_match)&.captures&.first,
            image_url: og_image_match&.captures&.first,
            site_name: og_site_match&.captures&.first
          )
        end
      end
    rescue => e
      Rails.logger.warn("Link preview fetch failed for #{url}: #{e.message}")
    end
  end
end
