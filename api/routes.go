package api

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	scheduled_tasks "github.com/igezt/chronix/internal/db"
	"github.com/igezt/chronix/internal/worker"
)

func SetupRoutes(app *fiber.App, client *asynq.Client, dbConn *sql.DB) {
	var validate = validator.New()

	app.Post("/schedule/reminder", func(c *fiber.Ctx) error {
		type Request struct {
			UserID             int    `json:"user_id"`
			Message            string `json:"message" validate:"required"`
			Email              string `json:"email" validate:"required,email"`
			RunAt              string `json:"run_at"` // ISO8601 timestamp
			ReccurenceInterval *int   `json:"reccurence_interval"`
			ReccurenceLimit    *int   `json:"reccurence_limit"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
		}

		if err := validate.Struct(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		runAt, err := time.Parse(time.RFC3339, req.RunAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid run_at format"})
		}

		// task, err := worker.NewEmailReminderTask(req.UserID, req.Email, req.Message)
		// if err != nil {
		// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create task"})
		// }

		// info, err := client.Enqueue(task, asynq.ProcessAt(runAt))
		// if err != nil {
		// 	log.Printf("Failed to enqueue task: %v", err)
		// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Queue error"})
		// }

		payload := map[string]any{
			"email":   req.Email,
			"message": req.Message,
		}

		taskID, err := scheduled_tasks.InsertScheduledTask(context.Background(), dbConn, scheduled_tasks.ScheduledTask{
			UserID:             req.UserID,
			TaskType:           worker.TypeEmailReminder,
			Payload:            payload,
			RunAt:              runAt.Add(time.Duration(*req.ReccurenceInterval)),
			RecurrenceLimit:    req.ReccurenceLimit,
			RecurrenceInterval: req.ReccurenceInterval,
		})

		if err != nil {
			log.Printf("Failed to enqueue task: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Queue error"})
		}

		return c.JSON(fiber.Map{
			"task_id": taskID,
			"run_at":  runAt.Format(time.RFC3339),
		})
	})

	app.Post("/schedule/reminder/recurring", func(c *fiber.Ctx) error {
		return c.SendString("Chronix is running")
	})

	// health check
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Chronix is running")
	})
}
