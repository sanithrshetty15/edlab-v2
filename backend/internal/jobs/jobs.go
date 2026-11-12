package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

// Handler processes the JSON payload of a queued job.
type Handler func(context.Context, json.RawMessage) error

// Manager handles durable background job processing.
type Manager struct {
	DB       *gorm.DB
	Config   *config.Config
	Log      *slog.Logger
	handlers map[string]Handler

	mu    sync.RWMutex
	wake  chan struct{}
	once  sync.Once
	wg    sync.WaitGroup
}

type claimedJob struct {
	ID          int64          `gorm:"column:id"`
	JobType     string         `gorm:"column:job_type"`
	Payload     datatypes.JSON `gorm:"column:payload"`
	Attempts    int            `gorm:"column:attempts"`
	MaxAttempts int            `gorm:"column:max_attempts"`
}

// New creates a new job manager.
func New(cfg *config.Config, db *gorm.DB, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.Default()
	}
	return &Manager{
		DB:       db,
		Config:   cfg,
		Log:      log,
		handlers: make(map[string]Handler),
		wake:     make(chan struct{}, 1),
	}
}

// Register registers a job type with its handler function.
func (j *Manager) Register(name string, handler Handler) {
	if name == "" || handler == nil {
		j.Log.Warn("ignoring invalid job registration", "type", name)
		return
	}
	j.mu.Lock()
	j.handlers[name] = handler
	j.mu.Unlock()
}

func (j *Manager) handler(name string) (Handler, bool) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	handler, ok := j.handlers[name]
	return handler, ok
}

// Enqueue persists a job for background processing.
func (j *Manager) Enqueue(ctx context.Context, name string, payload json.RawMessage, runAt time.Time) error {
	if j.DB == nil {
		return errors.New("job manager database is not configured")
	}
	if name == "" {
		return errors.New("job type is required")
	}
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	if !json.Valid(payload) {
		return errors.New("job payload must be valid JSON")
	}
	if runAt.IsZero() || runAt.Before(time.Now().UTC()) {
		runAt = time.Now().UTC()
	}
	job := models.BackgroundJob{
		JobType:     name,
		Payload:     datatypes.JSON(payload),
		Status:      "pending",
		MaxAttempts: 5,
		RunAt:       runAt,
	}
	if err := j.DB.WithContext(ctx).Create(&job).Error; err != nil {
		return fmt.Errorf("enqueue %s job: %w", name, err)
	}
	j.signal()
	return nil
}

// EnqueueMail queues an outbound email.
func (j *Manager) EnqueueMail(ctx context.Context, to, subject, htmlBody string) error {
	payload, err := json.Marshal(map[string]string{
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
	})
	if err != nil {
		return err
	}
	return j.Enqueue(ctx, "mail.send", payload, time.Now().UTC())
}

// Start launches the worker pool. It returns after all workers exit when ctx is cancelled.
func (j *Manager) Start(ctx context.Context, workers int) {
	if workers < 1 {
		workers = 1
	}
	j.once.Do(func() {
		j.recoverStale(ctx)
		for i := 0; i < workers; i++ {
			j.wg.Add(1)
			go j.worker(ctx)
		}
		j.Log.Info("background job worker started", "workers", workers)
	})
}

// Wait blocks until all workers have stopped.
func (j *Manager) Wait() {
	j.wg.Wait()
}

func (j *Manager) signal() {
	select {
	case j.wake <- struct{}{}:
	default:
	}
}

func (j *Manager) worker(ctx context.Context) {
	defer j.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-j.wake:
		}

		for {
			processed, err := j.processOne(ctx)
			if err != nil {
				j.Log.Error("background job processing failed", "error", err)
				break
			}
			if !processed {
				break
			}
		}
	}
}

func (j *Manager) recoverStale(ctx context.Context) {
	result := j.DB.WithContext(ctx).Exec(`
		UPDATE background_jobs
		SET status = 'pending', locked_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE status = 'running' AND locked_at < CURRENT_TIMESTAMP - INTERVAL '15 minutes'
	`)
	if result.Error != nil {
		j.Log.Warn("failed to recover stale background jobs", "error", result.Error)
	} else if result.RowsAffected > 0 {
		j.Log.Warn("recovered stale background jobs", "count", result.RowsAffected)
	}
}

func (j *Manager) processOne(ctx context.Context) (bool, error) {
	var job claimedJob
	result := j.DB.WithContext(ctx).Raw(`
		WITH next AS (
			SELECT id FROM background_jobs
			WHERE status IN ('pending', 'retrying') AND run_at <= CURRENT_TIMESTAMP
			ORDER BY run_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE background_jobs b
		SET status = 'running', locked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		FROM next
		WHERE b.id = next.id
		RETURNING b.id, b.job_type, b.payload, b.attempts, b.max_attempts
	`).Scan(&job)
	if result.Error != nil {
		return false, result.Error
	}
	if job.ID == 0 {
		return false, nil
	}

	handler, ok := j.handler(job.JobType)
	if !ok {
		j.fail(ctx, job, fmt.Errorf("no handler registered for job type %q", job.JobType))
		return true, nil
	}

	jobCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	err := runHandler(jobCtx, handler, json.RawMessage(job.Payload))
	if err == nil {
		j.DB.WithContext(ctx).Model(&models.BackgroundJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status":       "completed",
			"completed_at": time.Now().UTC(),
			"locked_at":    nil,
			"last_error":   nil,
		})
		return true, nil
	}

	j.fail(ctx, job, err)
	return true, nil
}

func runHandler(ctx context.Context, handler Handler, payload json.RawMessage) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("job handler panic: %v", recovered)
		}
	}()
	return handler(ctx, payload)
}

func (j *Manager) fail(ctx context.Context, job claimedJob, jobErr error) {
	attempts := job.Attempts + 1
	message := jobErr.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	updates := map[string]any{
		"attempts":   attempts,
		"locked_at":  nil,
		"last_error": message,
	}
	if attempts >= job.MaxAttempts {
		updates["status"] = "failed"
		updates["completed_at"] = time.Now().UTC()
	} else {
		backoff := time.Duration(1<<uint(attempts)) * 30 * time.Second
		if backoff > time.Hour {
			backoff = time.Hour
		}
		updates["status"] = "retrying"
		updates["run_at"] = time.Now().UTC().Add(backoff)
	}
	if err := j.DB.WithContext(ctx).Model(&models.BackgroundJob{}).Where("id = ?", job.ID).Updates(updates).Error; err != nil {
		j.Log.Error("failed to update background job after failure", "job_id", job.ID, "error", err)
	}
	j.Log.Error("background job failed", "job_id", job.ID, "type", job.JobType, "attempts", attempts, "error", message)
}

// ProcessDefaultQueue processes due jobs synchronously. Useful for tests and one-off runs.
func (j *Manager) ProcessDefaultQueue(ctx context.Context) error {
	for {
		processed, err := j.processOne(ctx)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}
	}
}

// Process processes due jobs from the durable queue.
func (j *Manager) Process(ctx context.Context, queueName string) error {
	return j.ProcessDefaultQueue(ctx)
}

// Schedule is retained for compatibility; recurring jobs are not used by the platform.
func (j *Manager) Schedule(name string, handler Handler, cronExpression string) error {
	j.Register(name, handler)
	j.Log.Warn("recurring job scheduling is not enabled; job only registered", "type", name, "cron", cronExpression)
	return nil
}
