// Tideland Go Actor
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor

//--------------------
// IMPORTS
//--------------------

import (
	"context"
	"fmt"
	"time"
)

//--------------------
// CONFIGURATION
//--------------------

// Config contains all configuration options for an Actor.
type Config struct {
	// Context defines the lifetime of the Actor. If nil,
	// context.Background() will be used.
	Context context.Context

	// QueueCap defines the capacity of the action queue.
	// Must be positive, default is 256.
	QueueCap int

	// Recoverer is called when a panic occurs during action
	// execution. If nil, a default recoverer will be used
	// that wraps the panic value in an error.
	Recoverer Recoverer

	// Finalizer is called when the Actor stops. It receives
	// any error that caused the stop and can transform it.
	// If nil, a default finalizer will be used that returns
	// the error unchanged.
	Finalizer Finalizer

	// ActionTimeout defines a default timeout for all actions.
	// If set to 0 (default), no timeout is applied.
	// Can be overridden per action using DoSyncTimeout or DoAsyncTimeout.
	ActionTimeout time.Duration
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		Context:       context.Background(),
		QueueCap:      256,
		Recoverer:     defaultRecoverer,
		Finalizer:     defaultFinalizer,
		ActionTimeout: 0, // no default timeout
	}
}

// Validate checks if the configuration is valid and
// sets default values where needed.
func (c *Config) Validate() error {
	// Set defaults for nil values.
	if c.Context == nil {
		c.Context = context.Background()
	}
	if c.QueueCap < 1 {
		return NewError("Config.Validate", fmt.Errorf("queue capacity must be positive: %d", c.QueueCap), ErrInvalid)
	}
	if c.Recoverer == nil {
		c.Recoverer = defaultRecoverer
	}
	if c.Finalizer == nil {
		c.Finalizer = defaultFinalizer
	}
	return nil
}

// defaultRecoverer creates an error from a panic.
func defaultRecoverer(reason any) error {
	return NewError("Recover", fmt.Errorf("panic: %v", reason), ErrPanic)
}

// defaultFinalizer wraps any error in a shutdown error.
func defaultFinalizer(err error) error {
	if err != nil {
		return NewError("Finalize", err, ErrShutdown)
	}
	return nil
}

// EOF