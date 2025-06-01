package worker

const (
	TypeEmailReminder = "email:reminder"
)

type EmailReminderPayload struct {
	UserID  int    `json:"user_id"`
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
	Email   string `json:"email"`
}
