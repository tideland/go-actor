// Tideland Go Actor
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
// PERIODICAL
//--------------------

// PeriodicalTimeout runs an Action in a given interval. It will
// be done asynchronously with the given timeout. If the Actor
// is stopped, the periodical will be stopped, too. Starting the
// periodical also returns a function to stop only this periodical.
func (act *Actor) PeriodicalTimeout(
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
			case <-act.Done():
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

// Periodical runs an Action in a given interval. If the Actor is stopped,
// the periodical will be stopped, too. Starting the periodical also returns
// a function to stop only this periodical.
func (act *Actor) Periodical(
	interval time.Duration,
	action Action) (func(), error) {
	return act.PeriodicalTimeout(interval, action, defaultTimeout)
}

// EOF
