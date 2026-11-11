class Api::V1::TasksController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :require_admin!, only: [:create, :update, :destroy, :verify_submission]

  # GET /api/v1/tasks
  def index
    scope = Task
      .where(deleted_at: nil)
      .order(deadline: :desc, published_at: :desc)

    if params[:upcoming].present? && params[:upcoming] == "true"
      scope = scope.where("deadline >= ?", Date.current)
    end

    if params[:overdue].present? && params[:overdue] == "true"
      scope = scope.where("deadline < ?", Date.current)
    end

    if params[:task_type].present?
      scope = scope.where(task_type: params[:task_type])
    end

    member = current_user.member if current_user.member? && current_user.member.status == "active"

    if member && !current_user.admin?
      targeted_task_ids = TaskTarget.where(academic_year: member.academic_year).select(:task_id)
      scope = scope.where(id: targeted_task_ids)
    end

    tasks = paginate(scope.includes(:created_by))

    render json: {
      success: true,
      tasks: tasks.items.map { |t| serialize_task(t, member: member) },
      pagination: {
        page: tasks.current_page,
        per_page: tasks.per_page,
        total_count: tasks.total_count
      }
    }, status: :ok
  end

  # GET /api/v1/tasks/:id
  def show
    task = Task.find_by(id: params[:id], deleted_at: nil)

    unless task
      return render json: {
        success: false,
        message: "Task not found"
      }, status: :not_found
    end

    member = current_user.member if current_user.member? && current_user.member.status == "active"

    submission = nil
    if member
      submission = TaskSubmissionStatus.find_by(task_id: task.id, member_id: member.id)
    end

    render json: {
      success: true,
      task: serialize_task_full(task, member: member, submission: submission)
    }, status: :ok
  end

  # POST /api/v1/tasks
  def create
    title = params[:title].to_s.strip
    description = params[:description].to_s.strip
    task_type = params[:task_type].to_s.downcase
    deadline = params[:deadline].present? ? Date.parse(params[:deadline]) : nil
    instructions = params[:instructions].to_s.strip
    external_link = params[:external_link].to_s.strip
    required_attachments = params[:required_attachments].to_s.strip
    target_academic_years = params[:target_academic_years] || []

    validate_task_inputs(
      title: title,
      description: description,
      task_type: task_type,
      deadline: deadline,
      target_academic_years: target_academic_years
    )

    task = nil

    ActiveRecord::Base.transaction do
      task = Task.create!(
        title: title,
        description: description.presence,
        task_type: task_type,
        published_at: Time.current,
        deadline: deadline,
        instructions: instructions.presence,
        external_link: external_link.presence,
        required_attachments: required_attachments.presence,
        created_by: current_user
      )

      Array(target_academic_years).each do |ay|
        TaskTarget.create!(
          task_id: task.id,
          academic_year: ay.to_i
        )
      end

      target_members = Member.active
        .where(academic_year: target_academic_years.map(&:to_i))
        .pluck(:user_id)

      target_members.each do |user_id|
        TaskSubmissionStatus.create!(
          task_id: task.id,
          member_id: Member.find_by(user_id: user_id)&.id,
          status: "pending"
        )
      end

      AuditLogService.log_task_created(task, created_by: current_user)

      NotificationService.notify_task(task, created_by: current_user)
    end

    render json: {
      success: true,
      message: "Task created and published",
      task: serialize_task_full(task)
    }, status: :created
  end

  # PATCH /api/v1/tasks/:id
  def update
    task = Task.find_by(id: params[:id], deleted_at: nil)

    unless task
      return render json: {
        success: false,
        message: "Task not found"
      }, status: :not_found
    end

    changes = {}

    if params[:title].present? && task.title != params[:title].to_s.strip
      title = params[:title].to_s.strip
      unless title.present?
        return render json: {
          success: false,
          message: "Title cannot be empty"
        }, status: :unprocessable_entity
      end
      changes[:title] = { from: task.title, to: title }
      task.update!(title: title)
    end

    if params[:description].present? && task.description != params[:description].to_s.strip
      changes[:description] = { from: task.description.truncate(100), to: params[:description].truncate(100) }
      task.update!(description: params[:description].to_s.strip.presence)
    end

    if params[:deadline].present?
      deadline = Date.parse(params[:deadline])
      if task.deadline != deadline
        changes[:deadline] = { from: task.deadline.to_s, to: deadline.to_s }
        task.update!(deadline: deadline)
      end
    end

    if params[:instructions].present? && task.instructions != params[:instructions].to_s.strip
      changes[:instructions] = { from: task.instructions.truncate(100), to: params[:instructions].truncate(100) }
      task.update!(instructions: params[:instructions].to_s.strip.presence)
    end

    if params[:external_link].present? && task.external_link != params[:external_link].to_s.strip
      link = params[:external_link].to_s.strip
      unless link.blank? || link.match?(/\Ahttps?:\/\//i)
        return render json: {
          success: false,
          message: "External link must be a valid URL"
        }, status: :unprocessable_entity
      end
      changes[:external_link] = { from: task.external_link, to: link }
      task.update!(external_link: link.presence)
    end

    if params[:target_academic_years].present?
      new_targets = Array(params[:target_academic_years]).map(&:to_i).uniq
      current_targets = task.task_targets.pluck(:academic_year)

      if new_targets.sort != current_targets.sort
        task.task_targets.destroy_all
        new_targets.each do |ay|
          TaskTarget.create!(task_id: task.id, academic_year: ay)
        end
        changes[:target_academic_years] = { from: current_targets, to: new_targets }
      end
    end

    if changes.any?
      AuditLogService.log!(
        user: current_user,
        action: "task_updated",
        resource_type: "Task",
        resource_id: task.id,
        details: changes,
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "Task updated",
        task: serialize_task_full(task)
      }, status: :ok
    else
      render json: {
        success: false,
        message: "No changes provided"
      }, status: :unprocessable_entity
    end
  end

  # DELETE /api/v1/tasks/:id
  def destroy
    task = Task.find_by(id: params[:id], deleted_at: nil)

    unless task
      return render json: {
        success: false,
        message: "Task not found"
      }, status: :not_found
    end

    task.update!(deleted_at: Time.current)

    AuditLogService.log!(
      user: current_user,
      action: "task_deleted",
      resource_type: "Task",
      resource_id: task.id,
      details: { title: task.title, task_type: task.task_type },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "Task deleted"
    }, status: :ok
  end

  # POST /api/v1/tasks/:id/submit
  def submit
    task = Task.find_by(id: params[:id], deleted_at: nil)

    unless task
      return render json: {
        success: false,
        message: "Task not found"
      }, status: :not_found
    end

    member = current_user.member
    raise "Member profile required" unless member
    raise "Member not active" unless member.status == "active"

    submission = TaskSubmissionStatus.find_by(task_id: task.id, member_id: member.id)

    unless submission
      return render json: {
        success: false,
        message: "No submission record found for this task"
      }, status: :not_found
    end

    if submission.status == "verified"
      return render json: {
        success: false,
        message: "This submission has already been verified"
      }, status: :conflict
    end

    submission.update!(
      status: "submitted",
      submitted_at: Time.current
    )

    AuditLogService.log_task_submission(submission, submitted_by: current_user)

    NotificationService.notify_task_submission(submission, submitted_by: current_user)

    render json: {
      success: true,
      message: "Task submitted successfully",
      submission: serialize_submission(submission)
    }, status: :ok
  end

  # POST /api/v1/tasks/:id/verify
  def verify_submission
    task = Task.find_by(id: params[:id], deleted_at: nil)

    unless task
      return render json: {
        success: false,
        message: "Task not found"
      }, status: :not_found
    end

    member_id = params[:member_id].to_i
    member = Member.find_by(id: member_id)

    unless member
      return render json: {
        success: false,
        message: "Member not found"
      }, status: :not_found
    end

    submission = TaskSubmissionStatus.find_by(task_id: task.id, member_id: member_id)

    unless submission
      return render json: {
        success: false,
        message: "Submission not found"
      }, status: :not_found
    end

    unless submission.status == "submitted"
      return render json: {
        success: false,
        message: "Submission is not in pending state"
      }, status: :unprocessable_entity
    end

    action = (params[:decision].presence || params[:verification_action]).to_s.downcase

    unless %w[verify reject].include?(action)
      return render json: {
        success: false,
        message: "Action must be 'verify' or 'reject'"
      }, status: :unprocessable_entity
    end

    submission.update!(
      status: action == "verify" ? "verified" : "rejected",
      verified_at: Time.current,
      verified_by: current_user
    )

    AuditLogService.log_task_verified(submission, verified_by: current_user)

    NotificationService.notify_task_verified(submission, verified_by: current_user)

    render json: {
      success: true,
      message: action == "verify" ? "Submission verified" : "Submission rejected",
      submission: serialize_submission(submission)
    }, status: :ok
  end

  private

  def require_admin!
    unless current_user.admin?
      return render json: {
        success: false,
        message: "Administrator access required"
      }, status: :forbidden
    end
  end

  def validate_task_inputs(title:, description:, task_type:, deadline:, target_academic_years:)
    raise "Title is required" if title.blank?
    raise "Task type is required" if task_type.blank?
    raise "Deadline is required" if deadline.blank?
    raise "At least one target academic year is required" if target_academic_years.blank?

    unless %w[assignment project report activity other].include?(task_type)
      raise "Invalid task type"
    end

    target_academic_years.each do |ay|
      unless ay.to_i.between?(1, 8)
        raise "Academic year must be between 1 and 8"
      end
    end
  end


  def serialize_task(task, member: nil)
    submission = member ? TaskSubmissionStatus.find_by(task_id: task.id, member_id: member.id) : nil

    {
      id: task.id,
      title: task.title,
      task_type: task.task_type,
      deadline: task.deadline.to_s,
      instructions: task.instructions,
      external_link: task.external_link,
      required_attachments: task.required_attachments,
      created_by: {
        id: task.created_by.id,
        name: task.created_by.admin&.name || task.created_by.email
      },
      target_academic_years: task.task_targets.pluck(:academic_year),
      created_at: task.created_at,
      published_at: task.published_at&.iso8601,
      submission_status: submission&.status,
      is_submitted: submission&.status == "submitted",
      is_verified: submission&.status == "verified"
    }
  end

  def serialize_task_full(task, member: nil, submission: nil)
    serialize_task(task, member: member).merge(
      description: task.description,
      submission: submission ? serialize_submission(submission) : nil
    )
  end

  def serialize_submission(submission)
    {
      id: submission.id,
      task_id: submission.task_id,
      member_id: submission.member_id,
      status: submission.status,
      submitted_at: submission.submitted_at,
      verified_at: submission.verified_at,
      verified_by: submission.verified_by&.email
    }
  end
end
