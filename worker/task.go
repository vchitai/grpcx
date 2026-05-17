package worker

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

// Task is a typed task definition. P is the JSON-serializable payload type.
// Declare one package-level variable per task type:
//
//	var SendEmail = worker.NewTask[SendEmailPayload]("email:send")
type Task[P any] struct {
	typeName string
}

// NewTask creates a new typed Task with the given type name.
func NewTask[P any](typeName string) Task[P] {
	return Task[P]{typeName: typeName}
}

// TypeName returns the asynq task type string.
func (t Task[P]) TypeName() string {
	return t.typeName
}

// New serializes payload to JSON and returns an *asynq.Task ready to enqueue.
func (t Task[P]) New(payload P, opts ...asynq.Option) (*asynq.Task, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(t.typeName, b, opts...), nil
}

// Handler returns an asynq.Handler that deserializes the task payload and
// calls fn with the typed value.
func (t Task[P]) Handler(fn func(ctx context.Context, payload P) error) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		var p P
		if err := json.Unmarshal(task.Payload(), &p); err != nil {
			return err
		}
		return fn(ctx, p)
	})
}
