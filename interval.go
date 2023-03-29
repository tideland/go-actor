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

// IntervalTimeout uses a given Actor to run a function in a
// given interval. It will be done asynchronously with the
// given timeout. If the Actor is stopped, IntervalTimeout
// will be stopped, too. Calling the IntervalTimeout returns
// a function to stop the interval.
func IntervalTimeout(
	act *Actor,
	interval time.Duration,
	action Action,
	timeout time.Duration) (func(), error) {
	if act.Err() != nil {
		return nil, act.Err()
	}
	stopc := make(chan struct{})
	stopper := func() {
		if stopc != nil {
			close(stopc)
		}
	}
	// Goroutine to run the interval.
	go func() {
		defer func() {
			stopc = nil
		}()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-act.ctx.Done():
				return
			case <-stopc:
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
func Interval(
	act *Actor,
	interval time.Duration,
	action Action) (func(), error) {
	return IntervalTimeout(act, interval, action, defaultTimeout)
}

// EOF
