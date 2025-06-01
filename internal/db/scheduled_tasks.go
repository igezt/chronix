package scheduled_tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

type ScheduledTask struct {
	ID                 string
	UserID             int
	TaskType           string
	Payload            json.RawMessage
	RunAt              time.Time
	Status             string
	RecurrenceInterval sql.NullInt64 // seconds
	RecurrenceLimit    sql.NullInt64
}

func InsertScheduledTask(ctx context.Context, db *sql.DB, task ScheduledTask) (string, error) {
	payloadBytes, err := json.Marshal(task.Payload)
	if err != nil {
		return "", err
	}

	var id string
	err = db.QueryRowContext(ctx, `
		INSERT INTO scheduled_tasks (user_id, task_type, payload, run_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		task.UserID, task.TaskType, payloadBytes, task.RunAt,
	).Scan(&id)

	return id, err
}

func ProcessRecurringScheduledTask(ctx context.Context, db *sql.DB, task ScheduledTask) error {
	if !task.RecurrenceInterval.Valid {
		log.Printf("Task %s is not recurring", task.ID)
		return nil
	}

	newRunAt := task.RunAt.Add(time.Duration(task.RecurrenceInterval.Int64) * time.Second)

	var (
		query string
		args  []any
	)

	if task.RecurrenceLimit.Valid {
		query = `
			UPDATE scheduled_tasks
			SET run_at = $1,
				recurrence_limit = recurrence_limit - 1,
				updated_at = now(),
				status = 'processing'
			WHERE id = $2`
		args = []any{newRunAt, task.ID}
	} else {
		query = `
			UPDATE scheduled_tasks
			SET run_at = $1,
				updated_at = now(),
				status = 'processing'
			WHERE id = $2`
		args = []any{newRunAt, task.ID}
	}

	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		log.Printf("Failed to reschedule recurring task %s: %v", task.ID, err)
		return err
	}

	return nil
}

func MarkCompleteScheduledTask(ctx context.Context, db *sql.DB, task ScheduledTask) error {

}
