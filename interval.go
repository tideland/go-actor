// Tideland Go Actor - Interval
//
// Copyright (C) 2019-2023 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor // import "tideland.dev/go/actor"

//--------------------
// IMPORTS
//--------------------

import (
	"time"
)

//--------------------
// INTERVAL
//--------------------

// IntervalTimeout runs an Action in agiven interval. It will
// be done asynchronously with the given timeout. If the Actor
// is stopped, IntervalTimeout will be stopped, too. Calling
// the IntervalTimeout returns a function to stop the interval.
func (act *Actor) IntervalTimeout(
	interval time.Duration,
	action Action,
	timeout time.Duration) (func(), error) {
	if act.Err() != nil {
		return nil, act.Err()
	}
	done := make(chan struct{})
	stopper := func() {
		if done != nil {
			close(done)
		}
	}
	// Goroutine to run the interval.
	go func() {
		defer func() {
			done = nil
		}()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-act.ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				if act.DoAsyncTimeout(action, timeout) != nil {
					return
				}
			}
		}
	}()
	return stopper, nil
}

// Interval uses a given Actor to run a function in a given interval.
// If the Actor is stopped, Interval will be stopped, too. Calling the
// Interval returns a function to stop the interval.
func (act *Actor) Interval(
	interval time.Duration,
	action Action) (func(), error) {
	return act.IntervalTimeout(interval, action, defaultTimeout)
}

// EOF
