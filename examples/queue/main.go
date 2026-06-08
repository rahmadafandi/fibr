// Copyright 2026 Rahmad Afandi. MIT License.

// Command queue demonstrates the asynq-backed job client and worker. Unlike the
// other examples it needs a running Redis: set REDIS_URL (default
// redis://localhost:6379). It enqueues one welcome-email task, then starts a
// worker that processes it and blocks until SIGINT/SIGTERM.
//
//	docker run -p 6379:6379 redis    # in another terminal
//	go run ./queue
package main

import (
	"context"
	"log"
	"os"

	"github.com/rahmadafandi/fiber-helpers/jobs"
)

// WelcomePayload is the JSON payload carried by the task.
type WelcomePayload struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
}

const taskWelcomeEmail = "email:welcome"

func main() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := jobs.RedisConnOpt(redisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}

	// Producer: enqueue a task. In a real app this lives in your HTTP handler.
	client := jobs.NewClient(opt)
	info, err := client.Enqueue(context.Background(), taskWelcomeEmail, WelcomePayload{
		UserID: 1,
		Email:  "ada@example.com",
	})
	if err != nil {
		log.Fatalf("enqueue: %v", err)
	}
	_ = client.Close()
	log.Printf("enqueued task id=%s queue=%s", info.ID, info.Queue)

	// Consumer: register the typed handler and run the worker (blocks).
	srv := jobs.NewServer(opt, jobs.ServerConfig{Concurrency: 10})
	jobs.Handle(srv, taskWelcomeEmail, func(_ context.Context, p WelcomePayload) error {
		log.Printf("sending welcome email to user %d at %s", p.UserID, p.Email)
		return nil
	})

	log.Println("worker started; press Ctrl+C to stop")
	if err := srv.Run(); err != nil {
		log.Fatalf("worker: %v", err)
	}
}
