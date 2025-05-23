// Tideland Go Actor
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor // import "tideland.dev/go/actor"

//--------------------
// IMPORTS
//--------------------

import (
	"context"
	"fmt"
)

//--------------------
// CONSTANTS
//--------------------

const (
	// defaultQueueCap is the minimum and default capacity
	// of the async actions queue.
	defaultQueueCap = 256
)

//--------------------
// OPTIONS
//--------------------

// Option defines a function applying an option to an Actor.
type Option func(*Actor) error

// WithContext sets the context of the Actor.
func WithContext(ctx context.Context) Option {
	return func(a *Actor) error {
		if ctx == nil {
			return NewError("WithContext", fmt.Errorf("context cannot be nil"), ErrInvalid)
		}
		a.ctx, a.cancel = context.WithCancel(ctx)
		return nil
	}
}

// WithQueueCap sets the capacity of the action queue.
func WithQueueCap(cap int) Option {
	return func(a *Actor) error {
		if cap < 1 {
			return NewError("WithQueueCap", fmt.Errorf("queue capacity must be positive: %d", cap), ErrInvalid)
		}
		a.requests = make(chan *request, cap)
		return nil
	}
}

// WithRecoverer sets the recoverer function.
func WithRecoverer(recoverer Recoverer) Option {
	return func(a *Actor) error {
		if recoverer == nil {
			return NewError("WithRecoverer", fmt.Errorf("recoverer cannot be nil"), ErrInvalid)
		}
		a.recoverer = recoverer
		return nil
	}
}

// WithFinalizer sets the finalizer function.
func WithFinalizer(finalizer Finalizer) Option {
	return func(a *Actor) error {
		if finalizer == nil {
			return NewError("WithFinalizer", fmt.Errorf("finalizer cannot be nil"), ErrInvalid)
		}
		a.finalizer = finalizer
		return nil
	}
}

// applyOptions applies the given options to the Actor.
func applyOptions(a *Actor, options ...Option) error {
	for _, option := range options {
		if err := option(a); err != nil {
			return err
		}
	}
	// Set defaults if not set by options.
	if a.ctx == nil {
		a.ctx, a.cancel = context.WithCancel(context.Background())
	}
	if a.requests == nil {
		a.requests = make(chan *request, defaultQueueCap)
	}
	if a.recoverer == nil {
		a.recoverer = defaultRecoverer
	}
	if a.finalizer == nil {
		a.finalizer = defaultFinalizer
	}
	return nil
}

// defaultRecoverer is the default recoverer function.
func defaultRecoverer(reason any) error {
	return NewError("Recover", fmt.Errorf("panic: %v", reason), ErrPanic)
}

// defaultFinalizer is the default finalizer function.
func defaultFinalizer(err error) error {
	if err != nil {
		return NewError("Finalize", err, ErrShutdown)
	}
	return nil
}

// EOF
