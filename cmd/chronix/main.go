package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"

	"github.com/igezt/chronix/api"
	"github.com/igezt/chronix/db"
	poller "github.com/igezt/chronix/internal/scheduler"
	"github.com/igezt/chronix/internal/worker"
)

func main() {

	err := godotenv.Load() // defaults to .env
	if err != nil {
		log.Fatal("No .env file found or failed to load it")
	}

	// Connect to DB
	database, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// Run migrations
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	// Load configuration (can replace with Viper or .env later)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Initialize Redis client for Asynq
	redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	asynqSrv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"default":  6,
				"critical": 4,
			},
		},
	)

	// Create a worker mux to handle tasks
	mux := asynq.NewServeMux()
	worker.RegisterHandlers(mux, database)

	// Start worker in background
	go func() {
		if err := asynqSrv.Run(mux); err != nil {
			log.Fatalf("Asynq server error: %v", err)
		}
	}()

	poller.Start(database, asynqClient, 5*time.Second)

	// Setup HTTP server
	app := fiber.New()
	api.SetupRoutes(app, asynqClient, database)

	// Run HTTP server in background
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	waitForShutdown(app, asynqClient)
}

func waitForShutdown(app *fiber.App, client *asynq.Client) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(); err != nil {
		log.Fatalf("Fiber shutdown failed: %v", err)
	}

	if err := client.Close(); err != nil {
		log.Printf("Failed to close Asynq client: %v", err)
	}

	log.Println("Chronix exited gracefully.")
}
