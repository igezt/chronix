package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hibiken/asynq"
	scheduled_tasks "github.com/igezt/chronix/internal/db"
	"github.com/igezt/chronix/internal/tools/mailer"
)

func RegisterHandlers(mux *asynq.ServeMux, db *sql.DB) {
	// Wrap handler so it has access to db
	mux.HandleFunc(TypeEmailReminder, func(ctx context.Context, t *asynq.Task) error {
		return handleEmailReminder(ctx, t, db)
	})
}

func handleEmailReminder(ctx context.Context, t *asynq.Task, db *sql.DB) error {

	updatedStatus := "pending"

	var task scheduled_tasks.ScheduledTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to parse task: %w", err)
	}

	isProcessing, isProcessingErr := scheduled_tasks.IsTaskProcessing(ctx, db, task)
	if isProcessingErr != nil {
		return fmt.Errorf("failed to check if task is still processing: %w", isProcessingErr)
	}

	if !isProcessing {
		log.Printf("task %s is not of processing status, early exit", task.ID)
		return fmt.Errorf("task %s is not of processing status, early exit", task.ID)
	}

	defer updateTaskStatus(ctx, db, task, updatedStatus, time.Now())

	var payload EmailReminderPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		log.Printf("failed to parse payload %s", err)
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	log.Printf("[Chronix] Sending reminder to user %d: %s\n", payload.UserID, payload.Message)

	err := mailer.SendEmail(
		os.Getenv("SENDER_EMAIL"),
		payload.Email,
		"Reminder",
		payload.Message,
		"smtp.gmail.com",
		587,
		os.Getenv("SENDER_EMAIL"),
		os.Getenv("SMTP_PASSWORD"),
	)
	updatedStatus = "completed"

	if err != nil {
		log.Printf("Failed to send email: %v", err)
	}
	return nil
}

func updateTaskStatus(ctx context.Context, db *sql.DB, task scheduled_tasks.ScheduledTask, updatedStatus string, runTime time.Time) {
	scheduled_tasks.CompleteScheduledTask(ctx, db, task, updatedStatus, runTime)
}
