// Tideland Go Actor
//
// Copyright (C) 2019-2021 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor // import "tideland.dev/go/actor"

//--------------------
// IMPORTS
//--------------------

import (
	"context"
	"time"
)

//--------------------
// OPTIONS
//--------------------

// Option defines the signature of an option setting function.
type Option func(act *Actor) error

// WithContext allows to pass a context for cancellation or timeout.
func WithContext(ctx context.Context) Option {
	return func(act *Actor) error {
		act.ctx = ctx
		return nil
	}
}

// WithQueueCap defines the channel capacity for actions sent to an Actor.
func WithQueueCap(c int) Option {
	return func(act *Actor) error {
		if c < defaultQueueCap {
			c = defaultQueueCap
		}
		act.asyncActions = make(chan Action, c)
		return nil
	}
}

// WithTimeout sets the timeout for sending actions to the actor.
func WithTimeout(timeout time.Duration) Option {
	return func(act *Actor) error {
		if timeout < 0 {
			timeout = defaultTimeout
		}
		act.timeout = timeout
		return nil
	}
}

// WithRepairer defines the panic handler of an actor.
func WithRepairer(repairer Repairer) Option {
	return func(act *Actor) error {
		act.repairer = repairer
		return nil
	}
}

// WithFinalizer sets a function for finalizing the
// work of a Loop.
func WithFinalizer(finalizer Finalizer) Option {
	return func(act *Actor) error {
		act.finalizer = finalizer
		return nil
	}
}

// EOF
