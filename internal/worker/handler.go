package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

const (
	TypeEmailReminder = "email:reminder"
)

type ReminderPayload struct {
	UserID  int    `json:"user_id"`
	Message string `json:"message"`
}

func RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEmailReminder, handleEmailReminder)
}

func handleEmailReminder(ctx context.Context, t *asynq.Task) error {
	var payload ReminderPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	log.Printf("[Chronix] Sending reminder to user %d: %s\n", payload.UserID, payload.Message)

	// TODO: replace this with real logic (send email, etc)
	return nil
}

// NewEmailReminderTask creates a new task for sending a reminder
func NewEmailReminderTask(userID int, message string) (*asynq.Task, error) {
	payload, err := json.Marshal(ReminderPayload{
		UserID:  userID,
		Message: message,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return asynq.NewTask(TypeEmailReminder, payload), nil
}
