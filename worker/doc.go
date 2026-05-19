// Package worker wraps github.com/hibiken/asynq to provide an ergonomic,
// type-safe API for defining, enqueueing, processing, and scheduling
// background tasks backed by Redis.
//
// # Defining tasks
//
// Declare one package-level variable per task type using a concrete payload
// struct. Keep these in a shared "tasks" package so both the enqueueing side
// and the handler side can import them without creating a cycle.
//
//	package tasks
//
//	import "github.com/vchitai/grpcx/worker"
//
//	type SendEmailPayload struct {
//	    To      string
//	    Subject string
//	    Body    string
//	}
//
//	var SendEmail = worker.NewTask[SendEmailPayload]("email:send")
//
//	type CleanupPayload struct{}
//
//	var Cleanup = worker.NewTask[CleanupPayload]("db:cleanup")
//
// # Enqueueing from a service
//
//	import (
//	    "context"
//
//	    "github.com/vchitai/grpcx/worker"
//	    "github.com/mycollectibles/mycollectibles-loan-backend/internal/tasks"
//	)
//
//	type UserService struct {
//	    queue *worker.Client
//	}
//
//	func (s *UserService) Register(ctx context.Context, email string) error {
//	    _, err := worker.Enqueue(ctx, s.queue, tasks.SendEmail, tasks.SendEmailPayload{
//	        To:      email,
//	        Subject: "Welcome!",
//	        Body:    "Thanks for signing up.",
//	    })
//	    return err
//	}
//
// # Registering handlers in the worker command
//
//	package handlers
//
//	import (
//	    "context"
//	    "log/slog"
//
//	    "github.com/mycollectibles/mycollectibles-loan-backend/internal/tasks"
//	)
//
//	func HandleSendEmail(ctx context.Context, p tasks.SendEmailPayload) error {
//	    slog.Info("sending email", "to", p.To, "subject", p.Subject)
//	    // ... send email via SMTP / SES / etc.
//	    return nil
//	}
//
//	// In cmd/worker/command.go:
//	//
//	//   worker.Register(srv, tasks.SendEmail, handlers.HandleSendEmail)
//	//   return srv.Run()
//
// # Scheduling periodic tasks in the cron command
//
//	// In cmd/cron/command.go:
//	//
//	//   sched := worker.NewScheduler(redisCfg)
//	//   worker.Schedule(sched, "@every 1h", tasks.Cleanup, tasks.CleanupPayload{})
//	//   sched.Start()
//	//   // ... wait for signal, then sched.Stop()
package worker
