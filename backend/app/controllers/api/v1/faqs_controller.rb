class Api::V1::FaqsController < Api::V1::BaseController
  before_action :authenticate_user!
  before_action :require_password_change_completed!
  before_action :require_admin!, only: [:create, :update, :destroy, :reorder]

  # GET /api/v1/faqs
  def index
    categories = FaqCategory
      .order(position: :asc)
      .includes(:faqs)

    render json: {
      success: true,
      faqs: categories.map { |cat|
        {
          id: cat.id,
          name: cat.name,
          slug: cat.slug,
          position: cat.position,
          faqs: cat.faqs.order(position: :asc).map { |faq|
            {
              id: faq.id,
              question: faq.question,
              answer: faq.answer,
              position: faq.position
            }
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/faqs/:slug
  def show_by_slug
    category = FaqCategory.find_by(slug: params[:slug])

    unless category
      return render json: {
        success: false,
        message: "FAQ category not found"
      }, status: :not_found
    end

    render json: {
      success: true,
      category: {
        id: category.id,
        name: category.name,
        slug: category.slug,
        position: category.position,
        faqs: category.faqs.order(position: :asc).map { |faq|
          {
            id: faq.id,
            question: faq.question,
            answer: faq.answer,
            position: faq.position
          }
        }
      }
    }, status: :ok
  end

  # GET /api/v1/faqs/categories
  def categories
    render json: {
      success: true,
      categories: FaqCategory.order(position: :asc).map { |c|
        {
          id: c.id,
          name: c.name,
          slug: c.slug,
          position: c.position,
          faq_count: c.faqs.count
        }
      }
    }, status: :ok
  end

  # POST /api/v1/faqs/categories
  def create_category
    name = params[:name].to_s.strip
    slug = params[:slug].to_s.strip.downcase.gsub(/[^a-z0-9-]/, "-")

    unless name.present?
      return render json: {
        success: false,
        message: "Name is required"
      }, status: :unprocessable_entity
    end

    unless slug.present?
      slug = name.downcase.gsub(/[^a-z0-9\s]/, "").gsub(/\s+/, "-")
    end

    if FaqCategory.exists?(slug: slug)
      return render json: {
        success: false,
        message: "A category with this slug already exists"
      }, status: :conflict
    end

    category = FaqCategory.create!(
      name: name,
      slug: slug,
      position: FaqCategory.maximum(:position).to_i + 1
    )

    render json: {
      success: true,
      message: "FAQ category created",
      category: serialize_category(category)
    }, status: :created
  end

  # POST /api/v1/faqs
  def create
    category_id = params[:category_id].to_i
    question = params[:question].to_s.strip
    answer = params[:answer].to_s.strip

    unless category_id.present?
      return render json: {
        success: false,
        message: "Category ID is required"
      }, status: :unprocessable_entity
    end

    unless question.present?
      return render json: {
        success: false,
        message: "Question is required"
      }, status: :unprocessable_entity
    end

    unless answer.present?
      return render json: {
        success: false,
        message: "Answer is required"
      }, status: :unprocessable_entity
    end

    category = FaqCategory.find_by(id: category_id)
    unless category
      return render json: {
        success: false,
        message: "FAQ category not found"
      }, status: :not_found
    end

    faq = Faq.create!(
      faq_category_id: category_id,
      question: question,
      answer: answer,
      position: category.faqs.maximum(:position).to_i + 1
    )

    AuditLogService.log!(
      user: current_user,
      action: "faq_created",
      resource_type: "Faq",
      resource_id: faq.id,
      details: {
        question: question.truncate(100),
        category_name: category.name
      },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "FAQ created",
      faq: serialize_faq(faq)
    }, status: :created
  end

  # PATCH /api/v1/faqs/:id
  def update
    faq = Faq.find(params[:id])

    changes = {}

    if params[:question].present? && faq.question != params[:question].to_s.strip
      question = params[:question].to_s.strip
      unless question.present?
        return render json: {
          success: false,
          message: "Question cannot be empty"
        }, status: :unprocessable_entity
      end
      changes[:question] = { from: faq.question, to: question }
      faq.update!(question: question)
    end

    if params[:answer].present? && faq.answer != params[:answer].to_s.strip
      answer = params[:answer].to_s.strip
      unless answer.present?
        return render json: {
          success: false,
          message: "Answer cannot be empty"
        }, status: :unprocessable_entity
      end
      changes[:answer] = { from: faq.answer.truncate(100), to: answer.truncate(100) }
      faq.update!(answer: answer)
    end

    if changes.any?
      AuditLogService.log!(
        user: current_user,
        action: "faq_updated",
        resource_type: "Faq",
        resource_id: faq.id,
        details: changes,
        ip_address: request.remote_ip
      )

      render json: {
        success: true,
        message: "FAQ updated",
        faq: serialize_faq(faq)
      }, status: :ok
    else
      render json: {
        success: false,
        message: "No changes provided"
      }, status: :unprocessable_entity
    end
  end

  # DELETE /api/v1/faqs/:id
  def destroy
    faq = Faq.find(params[:id])

    faq.destroy!

    render json: {
      success: true,
      message: "FAQ deleted"
    }, status: :ok
  end

  # PATCH /api/v1/faqs/reorder
  def reorder
    category_id = params[:category_id].to_i
    positions = params[:positions]

    unless positions.is_a?(Array)
      return render json: {
        success: false,
        message: "Positions must be an array"
      }, status: :unprocessable_entity
    end

    category = FaqCategory.find_by(id: category_id)
    unless category
      return render json: {
        success: false,
        message: "Category not found"
      }, status: :not_found
    end

    ActiveRecord::Base.transaction do
      positions.each_with_index do |faq_id, index|
        faq = Faq.find(faq_id)
        faq.update!(faq_category_id: category_id, position: index + 1)
      end
    end

    AuditLogService.log!(
      user: current_user,
      action: "faqs_reordered",
      resource_type: "FaqCategory",
      resource_id: category_id,
      details: { faq_count: category.faqs.count },
      ip_address: request.remote_ip
    )

    render json: {
      success: true,
      message: "FAQs reordered"
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


  def serialize_category(category)
    {
      id: category.id,
      name: category.name,
      slug: category.slug,
      position: category.position,
      faq_count: category.faqs.count
    }
  end

  def serialize_faq(faq)
    {
      id: faq.id,
      question: faq.question,
      answer: faq.answer,
      position: faq.position,
      category: {
        id: faq.faq_category.id,
        name: faq.faq_category.name,
        slug: faq.faq_category.slug
      }
    }
  end
end
