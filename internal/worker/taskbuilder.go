package worker

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// NewEmailReminderTask creates a new task for sending a reminder
func NewEmailReminderTask(userID int, email string, message string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailReminderPayload{
		UserID:  userID,
		Email:   email,
		Message: message,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return asynq.NewTask(TypeEmailReminder, payload), nil
}
