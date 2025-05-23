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
	"time"
)

//--------------------
// REPEAT
//--------------------

// RepeatWithContext runs an Action in a given interval. It will
// be done asynchronously until the context is canceled or timeout, the
// returned stopper function is called or the Actor is stopped.
func (act *Actor) RepeatWithContext(
	ctx context.Context,
	interval time.Duration,
	action Action) (func(), error) {
	if act.Err() != nil {
		return nil, act.Err()
	}
	ctx, cancel := context.WithCancel(ctx)
	// Goroutine to run the interval.
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-act.Done():
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if act.DoAsyncWithContext(ctx, action) != nil {
					return
				}
			}
		}
	}()
	return cancel, nil
}

// Repeat runs an Action in a given interval. It will
// be done asynchronously until the returned stopper function
// is called or the Actor is stopped.
func (act *Actor) Repeat(
	interval time.Duration,
	action Action) (func(), error) {
	return act.RepeatWithContext(context.Background(), interval, action)
}

// EOF
