package poller

import (
	"context"
	"database/sql"
	"encoding/json"
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

	log.Printf("Polling for tasks...")

	ctx := context.Background()

	rows, err := db.Query(`
		SELECT id, user_id, task_type, payload, run_at, recurrence_interval, recurrence_limit, recurrence_count
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
			&task.RunAt, &task.RecurrenceInterval, &task.RecurrenceLimit, &task.RecurrenceCount,
		); err != nil {
			return err
		}

		taskPayload, payloadErr := json.Marshal(task)
		if payloadErr != nil {
			log.Printf("Failed to marshal updated payload for task %s: %v", task.ID, payloadErr)
			continue
		}

		// Enqueue the task
		asynqTask := asynq.NewTask(task.TaskType, taskPayload)
		if _, err := asynqClient.Enqueue(asynqTask, asynq.MaxRetry(0)); err != nil {
			log.Printf("Failed to enqueue task %s: %v", task.ID, err)
			continue
		}

		err = scheduled_tasks.ProcessScheduledTask(ctx, db, task)
		if err != nil {
			log.Printf("Failed to mark task %s: %v", task.ID, err)
		}

		log.Printf("Inserted task %s of type %s to the queue", task.ID, task.TaskType)
	}
	return nil
}
