package worker

import (
	"time"

	"github.com/hibiken/asynq"
)

// SchedulerOption configures a Scheduler.
type SchedulerOption func(*asynq.SchedulerOpts)

// WithSchedulerLocation sets the time zone used for cron expressions.
// Defaults to UTC when not provided.
func WithSchedulerLocation(loc *time.Location) SchedulerOption {
	return func(o *asynq.SchedulerOpts) {
		o.Location = loc
	}
}

// Scheduler runs tasks on a cron schedule.
type Scheduler struct {
	s *asynq.Scheduler
}

// NewScheduler creates a new Scheduler connected to Redis using cfg.
// Scheduler logs are bridged to slog via asynqLogger.
func NewScheduler(cfg RedisConfig, opts ...SchedulerOption) *Scheduler {
	sopts := &asynq.SchedulerOpts{
		Logger: newAsynqLogger(),
	}
	for _, o := range opts {
		o(sopts)
	}

	s := asynq.NewScheduler(cfg.clientOpt(), sopts)
	return &Scheduler{s: s}
}

// NewSchedulerFromConfig creates a Scheduler from a YAML-mapped SchedulerConfig.
// If cfg.Location is empty, UTC is used.
func NewSchedulerFromConfig(redis RedisConfig, cfg SchedulerConfig, opts ...SchedulerOption) *Scheduler {
	var derived []SchedulerOption
	if cfg.Location != "" {
		loc, err := time.LoadLocation(cfg.Location)
		if err == nil {
			derived = append(derived, WithSchedulerLocation(loc))
		}
	}
	return NewScheduler(redis, append(derived, opts...)...)
}

// Schedule registers a periodic task using a cron expression and returns the
// entry ID assigned by the scheduler. It is a package-level function to work
// around Go's restriction on generic methods on non-generic types.
func Schedule[P any](s *Scheduler, cronExpr string, task Task[P], payload P, opts ...asynq.Option) (string, error) {
	t, err := task.New(payload, opts...)
	if err != nil {
		return "", err
	}
	return s.s.Register(cronExpr, t)
}

// Start starts the scheduler in the background (non-blocking).
func (s *Scheduler) Start() error {
	return s.s.Start()
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() {
	s.s.Shutdown()
}
