class MessageReplyService
  MAX_CONTENT_LENGTH = 10_000

  class ReplyError < StandardError; end

  def self.reply(message_id:, params:, user:, ip_address: nil)
    parent_message = GroupMessage.find_by(id: message_id)
    unless parent_message
      return { success: false, error: "Parent message not found", status: :not_found }
    end

    content = params[:content].to_s.strip

    unless content.present?
      return { success: false, error: "Reply content cannot be empty", status: :unprocessable_entity }
    end

    if content.length > MAX_CONTENT_LENGTH
      return { success: false, error: "Reply content is too long", status: :unprocessable_entity }
    end

    unless parent_message.is_deleted == false
      return { success: false, error: "Cannot reply to a deleted message", status: :gone }
    end

    new_message = GroupMessage.create!(
      user_id: user.id,
      content: content,
      message_type: "text",
      reply_to_id: parent_message.id,
      is_edited: false,
      is_deleted: false,
      is_pinned: false
    )

    {
      success: true,
      message: new_message
    }
  rescue => e
    Rails.logger.error("Message reply failed: #{e.class} - #{e.message}")
    {
      success: false,
      error: "Failed to create reply",
      status: :internal_server_error
    }
  end
end
