package worker

import (
	"context"

	"github.com/hibiken/asynq"
)

// Client enqueues tasks into the Redis-backed task queue.
type Client struct {
	c *asynq.Client
}

// NewClient creates a new Client connected to Redis using cfg.
func NewClient(cfg RedisConfig) *Client {
	return &Client{c: asynq.NewClient(cfg.clientOpt())}
}

// Close releases the underlying Redis connection.
func (c *Client) Close() error {
	return c.c.Close()
}

// Enqueue serializes payload, enqueues the task to Redis, and returns the
// resulting TaskInfo. It is a package-level function to work around Go's
// restriction on generic methods on non-generic types.
func Enqueue[P any](ctx context.Context, c *Client, task Task[P], payload P, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	t, err := task.New(payload, opts...)
	if err != nil {
		return nil, err
	}
	return c.c.EnqueueContext(ctx, t)
}
