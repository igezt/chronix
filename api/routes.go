package api

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/igezt/chronix/internal/worker"
)

func SetupRoutes(app *fiber.App, client *asynq.Client) {
	app.Post("/schedule/reminder", func(c *fiber.Ctx) error {
		type Request struct {
			UserID  int    `json:"user_id"`
			Message string `json:"message"`
			RunAt   string `json:"run_at"` // ISO8601 timestamp
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid body"})
		}

		runAt, err := time.Parse(time.RFC3339, req.RunAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid run_at format"})
		}

		task, err := worker.NewEmailReminderTask(req.UserID, req.Message)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create task"})
		}

		info, err := client.Enqueue(task, asynq.ProcessAt(runAt))
		if err != nil {
			log.Printf("Failed to enqueue task: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Queue error"})
		}

		return c.JSON(fiber.Map{
			"task_id": info.ID,
			"queue":   info.Queue,
			"run_at":  runAt.Format(time.RFC3339),
		})
	})

	// health check
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Chronix is running")
	})
}
