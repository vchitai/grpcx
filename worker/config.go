package worker

import "github.com/hibiken/asynq"

// RedisConfig holds the Redis connection config for the task queue.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// clientOpt converts RedisConfig into an asynq.RedisClientOpt that satisfies
// the asynq.RedisConnOpt interface.
func (c RedisConfig) clientOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	}
}

// ServerConfig is a YAML/mapstructure-compatible configuration for a worker Server.
// Zero values fall back to defaults (concurrency=10, standard queue priorities).
type ServerConfig struct {
	Concurrency int            `yaml:"concurrency" mapstructure:"concurrency"`
	Queues      map[string]int `yaml:"queues"      mapstructure:"queues"`
}

// SchedulerConfig is a YAML/mapstructure-compatible configuration for a Scheduler.
type SchedulerConfig struct {
	// Location is an IANA time zone name (e.g. "Asia/Ho_Chi_Minh"). Defaults to "UTC".
	Location string `yaml:"location" mapstructure:"location"`
}
