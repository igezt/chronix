package scheduled_tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

type ScheduledTask struct {
	ID                 string          `json:"id"`
	UserID             int             `json:"user_id"`
	TaskType           string          `json:"task_type"`
	Payload            json.RawMessage `json:"payload"`
	RunAt              time.Time       `json:"run_at"`
	Status             string          `json:"status"`
	RecurrenceInterval sql.NullInt64   `json:"recurrence_interval"`
	RecurrenceCount    int64           `json:"recurrence_count"`
	RecurrenceLimit    sql.NullInt64   `json:"recurrence_limit"`
}

func IsTaskProcessing(ctx context.Context, db *sql.DB, task ScheduledTask) (bool, error) {
	var status string
	err := db.QueryRow(`
		SELECT status FROM scheduled_tasks WHERE id = $1
	`, task.ID).Scan(&status)

	if err == sql.ErrNoRows {
		// Task doesn't exist
		return false, nil
	} else if err != nil {
		return false, err
	}

	return status == "processing", nil
}

func InsertScheduledTask(ctx context.Context, db *sql.DB, task ScheduledTask) (string, error) {
	payloadBytes, err := json.Marshal(task.Payload)
	if err != nil {
		return "", err
	}

	var id string
	err = db.QueryRowContext(ctx, `
		INSERT INTO scheduled_tasks (user_id, task_type, payload, run_at, recurrence_interval, recurrence_limit)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		task.UserID, task.TaskType, payloadBytes, task.RunAt, task.RecurrenceInterval, task.RecurrenceLimit,
	).Scan(&id)

	return id, err
}

func ProcessScheduledTask(ctx context.Context, db *sql.DB, task ScheduledTask) error {
	if !task.RecurrenceInterval.Valid {
		_, err := db.Exec(`
				UPDATE scheduled_tasks
				SET status = 'processing', updated_at = now()
				WHERE id = $1
			`, task.ID)
		return err
	}

	var (
		query string
		args  []any
	)

	query = `
		UPDATE scheduled_tasks
		SET updated_at = now(),
			status = 'processing'
		WHERE id = $1`
	args = []any{task.ID}
	// }

	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		log.Printf("Failed to reschedule recurring task %s: %v", task.ID, err)
		return err
	}

	return nil
}

func CompleteScheduledTask(ctx context.Context, db *sql.DB, task ScheduledTask, updatedStatus string, runTime time.Time) error {
	log.Printf("Updating task %s to %s", task.ID, updatedStatus)
	if !task.RecurrenceInterval.Valid {
		_, err := db.Exec(`
					UPDATE scheduled_tasks
					SET status = $2, updated_at = now(), run_at = $3
					WHERE id = $1
				`, task.ID, updatedStatus, runTime)
		return err
	}

	newRunAt := runTime.Add(time.Duration(task.RecurrenceInterval.Int64) * time.Second)

	if task.RecurrenceLimit.Valid {
		if task.RecurrenceLimit.Int64-task.RecurrenceCount <= 1 {
			_, err := db.Exec(`
				UPDATE scheduled_tasks
				SET status = 'completed', updated_at = now(), recurrence_count = recurrence_count + 1, run_at = $2
				WHERE id = $1
			`, task.ID, runTime)
			return err
		}
		_, err := db.Exec(`
				UPDATE scheduled_tasks
				SET run_at = $2, status = 'pending', updated_at = now(), recurrence_count = recurrence_count + 1
				WHERE id = $1
			`, task.ID, newRunAt)
		return err
	} else {
		_, err := db.Exec(`
				UPDATE scheduled_tasks
				SET status = 'pending', updated_at = now(), run_at = $2
				WHERE id = $1
			`, task.ID, newRunAt)
		return err
	}
}
