package poller

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/hibiken/asynq"
	scheduled_tasks "github.com/igezt/chronix/internal/db"
)

func Start(db *sql.DB, asynqClient *asynq.Client, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	go func() {
		for range ticker.C {
			if err := pollAndEnqueue(db, asynqClient); err != nil {
				log.Printf("Task poller error: %v", err)
			}
		}
	}()
	log.Println("[Poller] Started task poller")
}

func pollAndEnqueue(db *sql.DB, asynqClient *asynq.Client) error {
	ctx := context.Background()

	rows, err := db.Query(`
		SELECT id, user_id, task_type, payload, run_at, recurrence_interval, recurrence_limit
		FROM scheduled_tasks
		WHERE status = 'pending' AND run_at <= now()
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var task scheduled_tasks.ScheduledTask
		if err := rows.Scan(
			&task.ID, &task.UserID, &task.TaskType, &task.Payload,
			&task.RunAt, &task.RecurrenceInterval, &task.RecurrenceLimit,
		); err != nil {
			return err
		}

		// Enqueue the task
		asynqTask := asynq.NewTask(task.TaskType, task.Payload)
		if _, err := asynqClient.Enqueue(asynqTask); err != nil {
			log.Printf("Failed to enqueue task %s: %v", task.ID, err)
			continue
		}

		// Update the task: either mark as complete or reschedule
		// Task is a recurring task
		if task.RecurrenceInterval.Valid {
			scheduled_tasks.ProcessRecurringScheduledTask(ctx, db, task)
		} else {
			// Mark as complete
			_, err := db.Exec(`
				UPDATE scheduled_tasks
				SET status = 'complete', updated_at = now()
				WHERE id = $1
			`, task.ID)
			if err != nil {
				log.Printf("Failed to mark task %s complete: %v", task.ID, err)
			}
		}

	}
	return nil
}
