package config

import (
	"fmt"
	"time"
)

// Task runner timing defaults. Two invariants matter more than the individual
// numbers:
//
//	LeaseSeconds >= 3 * HeartbeatSeconds  — tighter and one slow disk write or
//	                                        GC pause causes a spurious reassignment
//	NoProgressSeconds > StepTimeoutSeconds — otherwise a legitimately slow step
//	                                        is killed for being "stuck"
const (
	DefaultTaskLeaseSeconds       = 30
	DefaultTaskHeartbeatSeconds   = 10
	DefaultTaskStepTimeoutSeconds = 120
	DefaultTaskNoProgressSeconds  = 300
	DefaultTaskWatchdogSeconds    = 10
	DefaultTaskIdleDelaySeconds   = 2
	DefaultTaskMaxSteps           = 100
	DefaultTaskMaxConcurrent      = 2
	DefaultTaskMaxAttempts        = 3
	DefaultTaskRetentionDays      = 30
)

// Normalize fills zero values with defaults and clamps the result into a
// coherent range. The two invariants are enforced rather than rejected here: an
// operator who tightens the lease should get a safe runner, not a dead one.
func (c *TasksConfig) Normalize() {
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = DefaultTaskMaxConcurrent
	}
	if c.LeaseSeconds <= 0 {
		c.LeaseSeconds = DefaultTaskLeaseSeconds
	}
	if c.HeartbeatSeconds <= 0 {
		c.HeartbeatSeconds = DefaultTaskHeartbeatSeconds
	}
	if c.StepTimeoutSeconds <= 0 {
		c.StepTimeoutSeconds = DefaultTaskStepTimeoutSeconds
	}
	if c.NoProgressSeconds <= 0 {
		c.NoProgressSeconds = DefaultTaskNoProgressSeconds
	}
	if c.WatchdogSeconds <= 0 {
		c.WatchdogSeconds = DefaultTaskWatchdogSeconds
	}
	if c.IdleDelaySeconds <= 0 {
		c.IdleDelaySeconds = DefaultTaskIdleDelaySeconds
	}
	if c.MaxSteps <= 0 {
		c.MaxSteps = DefaultTaskMaxSteps
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = DefaultTaskMaxAttempts
	}
	if c.RetentionDays <= 0 {
		c.RetentionDays = DefaultTaskRetentionDays
	}
	if c.LeaseSeconds < 3*c.HeartbeatSeconds {
		c.LeaseSeconds = 3 * c.HeartbeatSeconds
	}
	if c.NoProgressSeconds <= c.StepTimeoutSeconds {
		c.NoProgressSeconds = c.StepTimeoutSeconds * 2
	}
}

// Duration accessors for the runner. They assume Normalize has already run.
func (c TasksConfig) LeaseDuration() time.Duration {
	return time.Duration(c.LeaseSeconds) * time.Second
}
func (c TasksConfig) HeartbeatInterval() time.Duration {
	return time.Duration(c.HeartbeatSeconds) * time.Second
}
func (c TasksConfig) StepTimeout() time.Duration {
	return time.Duration(c.StepTimeoutSeconds) * time.Second
}
func (c TasksConfig) NoProgressWindow() time.Duration {
	return time.Duration(c.NoProgressSeconds) * time.Second
}
func (c TasksConfig) WatchdogInterval() time.Duration {
	return time.Duration(c.WatchdogSeconds) * time.Second
}
func (c TasksConfig) IdleDelay() time.Duration {
	return time.Duration(c.IdleDelaySeconds) * time.Second
}

func validateTasks(cfg TasksConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.MaxConcurrent < 1 || cfg.MaxConcurrent > 32 {
		return fmt.Errorf("tasks.max_concurrent must be between 1 and 32")
	}
	if cfg.HeartbeatSeconds < 1 || cfg.HeartbeatSeconds > 300 {
		return fmt.Errorf("tasks.heartbeat_seconds must be between 1 and 300")
	}
	if cfg.LeaseSeconds < 3*cfg.HeartbeatSeconds {
		return fmt.Errorf("tasks.lease_seconds must be at least 3x tasks.heartbeat_seconds")
	}
	if cfg.LeaseSeconds > 3600 {
		return fmt.Errorf("tasks.lease_seconds must not exceed 3600")
	}
	if cfg.StepTimeoutSeconds < 5 || cfg.StepTimeoutSeconds > 3600 {
		return fmt.Errorf("tasks.step_timeout_seconds must be between 5 and 3600")
	}
	if cfg.NoProgressSeconds <= cfg.StepTimeoutSeconds {
		return fmt.Errorf("tasks.no_progress_seconds must exceed tasks.step_timeout_seconds")
	}
	if cfg.NoProgressSeconds > 86400 {
		return fmt.Errorf("tasks.no_progress_seconds must not exceed 86400")
	}
	if cfg.WatchdogSeconds < 1 || cfg.WatchdogSeconds > 3600 {
		return fmt.Errorf("tasks.watchdog_seconds must be between 1 and 3600")
	}
	if cfg.IdleDelaySeconds < 1 || cfg.IdleDelaySeconds > 300 {
		return fmt.Errorf("tasks.idle_delay_seconds must be between 1 and 300")
	}
	if cfg.MaxSteps < 1 || cfg.MaxSteps > 1000 {
		return fmt.Errorf("tasks.max_steps must be between 1 and 1000")
	}
	if cfg.MaxAttempts < 1 || cfg.MaxAttempts > 20 {
		return fmt.Errorf("tasks.max_attempts must be between 1 and 20")
	}
	if cfg.RetentionDays < 1 || cfg.RetentionDays > 3650 {
		return fmt.Errorf("tasks.retention_days must be between 1 and 3650")
	}
	return nil
}
