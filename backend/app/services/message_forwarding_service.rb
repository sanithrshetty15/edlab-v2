class MessageForwardingService
  MAX_CONTENT_LENGTH = 10_000

  class ForwardError < StandardError; end

  def self.forward(original_message_id:, user:, ip_address: nil)
    original_message = GroupMessage.find_by(id: original_message_id)
    unless original_message
      return { success: false, error: "Original message not found", status: :not_found }
    end

    unless original_message.is_deleted == false
      return { success: false, error: "Cannot forward a deleted message", status: :gone }
    end

    ActiveRecord::Base.transaction do
      new_message = GroupMessage.create!(
        user_id: user.id,
        content: original_message.content,
        message_type: original_message.message_type,
        forwarded_from_id: original_message.id,
        is_edited: false,
        is_deleted: false,
        is_pinned: false
      )

      original_message.attachments.each do |attachment|
        MessageAttachment.create!(
          group_message_id: new_message.id,
          filename: attachment.filename,
          content_type: attachment.content_type,
          byte_size: attachment.byte_size,
          storage_key: attachment.storage_key
        )
      end

      create_receipts_for_message(new_message, user)

      if original_message.link_preview
        LinkPreview.create!(
          group_message_id: new_message.id,
          url: original_message.link_preview.url,
          title: original_message.link_preview.title,
          description: original_message.link_preview.description,
          image_url: original_message.link_preview.image_url,
          site_name: original_message.link_preview.site_name
        )
      end

      new_message
    end

    {
      success: true,
      message: new_message
    }
  rescue => e
    Rails.logger.error("Message forwarding failed: #{e.class} - #{e.message}")
    {
      success: false,
      error: "Failed to forward message",
      status: :internal_server_error
    }
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
end
