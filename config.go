package actor

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Config configures an Actor using fluent builder pattern.
// All fields are private and accessed via getters. Validation errors are
// accumulated and can be checked before creating the actor.
type Config struct {
	// Configuration fields
	ctx             context.Context
	queueCapacity   int
	actionTimeout   time.Duration
	shutdownTimeout time.Duration
	finalizer       Finalizer

	// Error accumulation
	err error
}

// NewConfig creates a new configuration with the given context.
// All other fields are set to sensible defaults.
func NewConfig(ctx context.Context) *Config {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Config{
		ctx:             ctx,
		queueCapacity:   256,
		actionTimeout:   0, // No timeout by default
		shutdownTimeout: 5 * time.Second,
		finalizer:       nil,
	}
}

// DefaultConfig creates a configuration with all default values.
func DefaultConfig() *Config {
	return NewConfig(context.Background())
}

// SetContext sets the context for the actor's lifecycle.
func (c *Config) SetContext(ctx context.Context) *Config {
	if ctx == nil {
		c.wrapError(fmt.Errorf("context cannot be nil"))
		return c
	}
	c.ctx = ctx
	return c
}

// SetQueueCapacity sets the maximum number of pending requests.
// Must be positive.
func (c *Config) SetQueueCapacity(capacity int) *Config {
	if capacity <= 0 {
		c.wrapError(fmt.Errorf("queue capacity must be positive, got %d", capacity))
		return c
	}
	c.queueCapacity = capacity
	return c
}

// SetActionTimeout sets the maximum time an action can run.
// Zero means no timeout. Negative values are rejected.
func (c *Config) SetActionTimeout(timeout time.Duration) *Config {
	if timeout < 0 {
		c.wrapError(fmt.Errorf("action timeout cannot be negative, got %v", timeout))
		return c
	}
	c.actionTimeout = timeout
	return c
}

// SetShutdownTimeout sets the maximum time to wait for graceful shutdown.
// Must be positive.
func (c *Config) SetShutdownTimeout(timeout time.Duration) *Config {
	if timeout <= 0 {
		c.wrapError(fmt.Errorf("shutdown timeout must be positive, got %v", timeout))
		return c
	}
	c.shutdownTimeout = timeout
	return c
}

// SetFinalizer sets a function to be called when the actor stops.
// The finalizer receives the error that caused the shutdown (if any).
func (c *Config) SetFinalizer(finalizer Finalizer) *Config {
	c.finalizer = finalizer
	return c
}

// Getters

// Context returns the configured context.
func (c *Config) Context() context.Context {
	return c.ctx
}

// QueueCapacity returns the configured queue capacity.
func (c *Config) QueueCapacity() int {
	return c.queueCapacity
}

// ActionTimeout returns the configured action timeout.
func (c *Config) ActionTimeout() time.Duration {
	return c.actionTimeout
}

// ShutdownTimeout returns the configured shutdown timeout.
func (c *Config) ShutdownTimeout() time.Duration {
	return c.shutdownTimeout
}

// Finalizer returns the configured finalizer function.
func (c *Config) Finalizer() Finalizer {
	return c.finalizer
}

// Error accumulation

// wrapError adds an error to the accumulated errors.
func (c *Config) wrapError(err error) {
	if c.err == nil {
		c.err = err
	} else {
		c.err = errors.Join(c.err, err)
	}
}

// Validate returns any accumulated validation errors.
// This is called automatically by Go(), but can be called earlier to check.
func (c *Config) Validate() error {
	return c.err
}

// Error is an alias for Validate() for consistency with worker package.
func (c *Config) Error() error {
	return c.err
}
