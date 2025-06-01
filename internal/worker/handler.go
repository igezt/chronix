package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/hibiken/asynq"
	"github.com/igezt/chronix/internal/tools/mailer"
)

func RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeEmailReminder, handleEmailReminder)
}

func handleEmailReminder(ctx context.Context, t *asynq.Task) error {
	var payload EmailReminderPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
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
	if err != nil {
		log.Printf("Failed to send email: %v", err)
	}
	return nil
}
