class Api::V1::MessagesController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :check_messaging_permissions!, only: [:create, :reply, :edit, :delete, :forward, :react, :pin, :star]
  before_action :set_message, only: [:show, :edit, :delete, :reply, :get_replies, :react, :remove_reaction, :get_reactions, :pin, :unpin, :get_receipts, :star, :unstar, :get_starred]

  # GET /api/v1/messages
  def index
    scope = GroupMessage
      .where(is_deleted: false)
      .order(created_at: :desc)
      .includes(:user, :reply_to, :forwarded_from)

    if params[:before].present?
      scope = scope.where("created_at < ?", Time.parse(params[:before]))
    end

    if params[:after].present?
      scope = scope.where("created_at > ?", Time.parse(params[:after]))
    end

    messages = paginate(scope)

    render json: {
      success: true,
      messages: messages.items.map { |m| serialize_message(m) },
      pagination: {
        page: messages.current_page,
        per_page: messages.per_page,
        total_count: messages.total_count,
        total_pages: messages.total_pages
      }
    }, status: :ok
  end

  # GET /api/v1/messages/:id
  def show
    render json: {
      success: true,
      message: serialize_message_full(@message)
    }, status: :ok
  end

  # POST /api/v1/messages
  def create
    result = MessageCreationService.create(
      params: message_params,
      user: current_user,
      ip_address: request.remote_ip
    )

    if result[:success]
      AuditLogService.log_message_sent(
        result[:message],
        sent_by: current_user
      )

      NotificationService.notify_message(
        message: result[:message],
        sender: current_user
      )

      render json: {
        success: true,
        message: serialize_message_full(result[:message])
      }, status: :created
    else
      render json: {
        success: false,
        message: result[:error]
      }, status: result[:status] || :unprocessable_entity
    end
  end

  # POST /api/v1/messages/:id/reply
  def reply
    result = MessageReplyService.reply(
      message_id: params[:id],
      params: reply_params,
      user: current_user,
      ip_address: request.remote_ip
    )

    if result[:success]
      NotificationService.notify_reply(
        message: result[:message],
        sender: current_user,
        parent_message: @message
      )

      render json: {
        success: true,
        message: serialize_message_full(result[:message])
      }, status: :created
    else
      render json: {
        success: false,
        message: result[:error]
      }, status: result[:status] || :unprocessable_entity
    end
  end

  # PATCH /api/v1/messages/:id
  def edit
    unless @message.user_id == current_user.id
      return render json: {
        success: false,
        message: "You can only edit your own messages"
      }, status: :forbidden
    end

    if @message.is_deleted
      return render json: {
        success: false,
        message: "Message has been deleted"
      }, status: :gone
    end

    new_content = params[:content].to_s.strip
    unless new_content.present?
      return render json: {
        success: false,
        message: "Message content cannot be empty"
      }, status: :unprocessable_entity
    end

    @message.update!(
      content: new_content,
      is_edited: true
    )

    AuditLogService.log!(
      user: current_user,
      action: "message_edited",
      resource_type: "GroupMessage",
      resource_id: @message.id,
      details: {
        previous_content_preview: @message.content.truncate(100)
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: serialize_message_full(@message)
    }, status: :ok
  end

  # DELETE /api/v1/messages/:id
  def delete
    unless @message.user_id == current_user.id || current_user.admin?
      return render json: {
        success: false,
        message: "You can only delete your own messages"
      }, status: :forbidden
    end

    was_deleted = @message.is_deleted

    @message.update!(is_deleted: true)

    if !was_deleted
      AuditLogService.log_message_deleted(
        @message,
        deleted_by: current_user
      )
    end

    render json: {
      success: true,
      message: "Message deleted successfully"
    }, status: :ok
  end

  # GET /api/v1/messages/:id/replies
  def get_replies
    replies = GroupMessage
      .where(reply_to_id: @message.id, is_deleted: false)
      .order(created_at: :asc)
      .includes(:user)

    render json: {
      success: true,
      replies: replies.map { |m| serialize_message(m) }
    }, status: :ok
  end

  # POST /api/v1/messages/:id/forward
  def forward
    forwarded_to_id = params[:forwarded_to_id].to_i

    unless forwarded_to_id.present?
      return render json: {
        success: false,
        message: "Forwarded message ID is required"
      }, status: :unprocessable_entity
    end

    original_message = GroupMessage.find_by(id: forwarded_to_id)
    unless original_message
      return render json: {
        success: false,
        message: "Original message not found"
      }, status: :not_found
    end

    result = MessageForwardingService.forward(
      original_message_id: forwarded_to_id,
      user: current_user,
      ip_address: request.remote_ip
    )

    if result[:success]
      NotificationService.notify_forward(
        message: result[:message],
        sender: current_user,
        original_message: original_message
      )

      render json: {
        success: true,
        message: serialize_message_full(result[:message])
      }, status: :created
    else
      render json: {
        success: false,
        message: result[:error]
      }, status: result[:status] || :unprocessable_entity
    end
  end

  # POST /api/v1/messages/:id/react
  def react
    emoji = params[:emoji].to_s

    unless emoji.present?
      return render json: {
        success: false,
        message: "Emoji is required"
      }, status: :unprocessable_entity
    end

    unless emoji.match?(/\A[\p{Emoji}]+\z/)
      return render json: {
        success: false,
        message: "Invalid emoji"
      }, status: :unprocessable_entity
    end

    reaction = MessageReaction.find_or_create_by!(
      group_message_id: @message.id,
      user_id: current_user.id,
      emoji: emoji
    )

    render json: {
      success: true,
      reaction: {
        id: reaction.id,
        emoji: reaction.emoji,
        group_message_id: reaction.group_message_id,
        user_id: reaction.user_id,
        created_at: reaction.created_at
      }
    }, status: :ok
  end

  # DELETE /api/v1/messages/:id/react
  def remove_reaction
    reaction = MessageReaction.find_by(
      group_message_id: @message.id,
      user_id: current_user.id,
      emoji: params[:emoji].to_s
    )

    unless reaction
      return render json: {
        success: false,
        message: "Reaction not found"
      }, status: :not_found
    end

    reaction.destroy!

    render json: {
      success: true,
      message: "Reaction removed"
    }, status: :ok
  end

  # GET /api/v1/messages/:id/reactions
  def get_reactions
    reactions = MessageReaction
      .where(group_message_id: @message.id)
      .includes(:user)
      .order(created_at: :asc)

    grouped = reactions.group_by(&:emoji).transform_values do |emoji_reactions|
      emoji_reactions.map { |r| { user_id: r.user_id, user_name: r.user.name, created_at: r.created_at } }
    end

    render json: {
      success: true,
      reactions: grouped
    }, status: :ok
  end

  # POST /api/v1/messages/:id/pin
  def pin
    unless current_user.admin?
      return render json: {
        success: false,
        message: "Only administrators can pin messages"
      }, status: :forbidden
    end

    if @message.is_pinned
      return render json: {
        success: false,
        message: "Message is already pinned"
      }, status: :unprocessable_entity
    end

    MessagePin.create!(
      group_message_id: @message.id,
      pinned_by_id: current_user.id
    )

    @message.update!(is_pinned: true)

    render json: {
      success: true,
      message: "Message pinned successfully"
    }, status: :ok
  end

  # DELETE /api/v1/messages/:id/pin
  def unpin
    unless current_user.admin?
      return render json: {
        success: false,
        message: "Only administrators can unpin messages"
      }, status: :forbidden
    end

    if !@message.is_pinned
      return render json: {
        success: false,
        message: "Message is not pinned"
      }, status: :unprocessable_entity
    end

    MessagePin.where(group_message_id: @message.id).destroy_all
    @message.update!(is_pinned: false)

    render json: {
      success: true,
      message: "Message unpinned"
    }, status: :ok
  end

  # GET /api/v1/messages/:id/receipts
  def get_receipts
    receipts = MessageReceipt
      .where(group_message_id: @message.id)
      .includes(:user)
      .order(read_at: :desc)
      .limit(50)

    render json: {
      success: true,
      receipts: receipts.map { |r|
        {
          user_id: r.user_id,
          user_name: r.user.name,
          delivered_at: r.delivered_at,
          read_at: r.read_at
        }
      },
      seen_by: receipts.where.not(read_at: nil).pluck(:user_id)
    }, status: :ok
  end

  # POST /api/v1/messages/:id/star
  def star
    star = MessageStar.create!(
      group_message_id: @message.id,
      user_id: current_user.id
    )

    render json: {
      success: true,
      star: {
        id: star.id,
        group_message_id: star.group_message_id,
        user_id: star.user_id,
        created_at: star.created_at
      }
    }, status: :created
  end

  # DELETE /api/v1/messages/:id/star
  def unstar
    star = MessageStar.find_by(
      group_message_id: @message.id,
      user_id: current_user.id
    )

    star&.destroy!

    render json: {
      success: true,
      message: "Message unstarred"
    }, status: :ok
  end

  # GET /api/v1/messages/:id/starred
  def get_starred
    stars = MessageStar
      .where(group_message_id: @message.id)
      .includes(:user)
      .order(created_at: :asc)

    render json: {
      success: true,
      starred_by: stars.map { |s|
        {
          user_id: s.user_id,
          user_name: s.user.name,
          starred_at: s.created_at
        }
      }
    }, status: :ok
  end

  # GET /api/v1/messages/search
  def search
    query = params[:q].to_s.strip

    unless query.present?
      return render json: {
        success: false,
        message: "Search query is required"
      }, status: :unprocessable_entity
    end

    results = GroupMessage
      .where(is_deleted: false)
      .where("content ILIKE ?", "%#{query}%")
      .order(created_at: :desc)
      .includes(:user)
      .limit(50)

    render json: {
      success: true,
      results: results.map { |m| serialize_message(m) }
    }, status: :ok
  end

  # GET /api/v1/messages/pinned
  def pinned_messages
    raise "Administrator access required" unless current_user.admin?

    pinned = GroupMessage
      .where(is_pinned: true, is_deleted: false)
      .order(created_at: :desc)
      .includes(:user)

    render json: {
      success: true,
      messages: pinned.map { |m| serialize_message(m) }
    }, status: :ok
  end

  private

  def check_messaging_permissions!
    return if current_user.admin?

    group_settings = GroupSetting.first
    return if group_settings.blank?

    if group_settings.permission == "admins_only"
      return render json: {
        success: false,
        message: "Messaging is currently disabled for members"
      }, status: :forbidden
    end
  end

  def set_message
    @message = GroupMessage.includes(:user).find(params[:id])
  rescue ActiveRecord::RecordNotFound
    render json: {
      success: false,
      message: "Message not found"
    }, status: :not_found
  end

  def message_params
    params.permit(:content, attachments: [:filename, :content_type, :byte_size, :storage_key])
  end

  def reply_params
    params.permit(:content)
  end


  def serialize_message(message)
    {
      id: message.id,
      user_id: message.user_id,
      user: {
        id: message.user.id,
        name: message.user.member&.name
      },
      content: message.content,
      message_type: message.message_type,
      reply_to_id: message.reply_to_id,
      forwarded_from_id: message.forwarded_from_id,
      is_edited: message.is_edited,
      is_deleted: message.is_deleted,
      is_pinned: message.is_pinned,
      created_at: message.created_at,
      updated_at: message.updated_at
    }
  end

  def serialize_message_full(message)
    serialize_message(message).merge(
      attachments: message.attachments.map { |a|
        {
          id: a.id,
          filename: a.filename,
          content_type: a.content_type,
          byte_size: a.byte_size,
          storage_key: a.storage_key,
          url: nil
        }
      },
      link_preview: message.link_preview,
      reactions_count: message.reactions.count,
      replies_count: GroupMessage.where(reply_to_id: message.id, is_deleted: false).count,
      stars_count: message.stars.count,
      receipts_count: message.receipts.count
    )
  end
end
