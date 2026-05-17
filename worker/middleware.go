package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// loggingMiddleware logs task start, completion duration, and any errors using
// slog. It reads the task ID and retry count from ctx via asynq helpers.
func loggingMiddleware(next asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		taskID, _ := asynq.GetTaskID(ctx)
		retryCount, _ := asynq.GetRetryCount(ctx)

		slog.InfoContext(ctx, "task started",
			"task_type", task.Type(),
			"task_id", taskID,
			"retry_count", retryCount,
		)

		start := time.Now()
		err := next.ProcessTask(ctx, task)
		elapsed := time.Since(start)

		if err != nil {
			slog.ErrorContext(ctx, "task failed",
				"task_type", task.Type(),
				"task_id", taskID,
				"retry_count", retryCount,
				"elapsed_ms", elapsed.Milliseconds(),
				"error", err,
			)
		} else {
			slog.InfoContext(ctx, "task completed",
				"task_type", task.Type(),
				"task_id", taskID,
				"retry_count", retryCount,
				"elapsed_ms", elapsed.Milliseconds(),
			)
		}

		return err
	})
}

// asynqLogger bridges asynq's Logger interface to slog.Default().
type asynqLogger struct{}

func newAsynqLogger() *asynqLogger {
	return &asynqLogger{}
}

func (l *asynqLogger) Debug(args ...any) {
	slog.Debug(fmt.Sprint(args...))
}

func (l *asynqLogger) Info(args ...any) {
	slog.Info(fmt.Sprint(args...))
}

func (l *asynqLogger) Warn(args ...any) {
	slog.Warn(fmt.Sprint(args...))
}

func (l *asynqLogger) Error(args ...any) {
	slog.Error(fmt.Sprint(args...))
}

func (l *asynqLogger) Fatal(args ...any) {
	slog.Error(fmt.Sprint(args...))
}
