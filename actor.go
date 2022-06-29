// Tideland Go Actor
//
// Copyright (C) 2019-2022 Frank Mueller / Tideland / Oldenburg / Germany
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
	"sync"
	"sync/atomic"
	"time"
)

//--------------------
// CONSTANTS
//--------------------

const (
	// defaultTimeout is used in a DoSync() call.
	defaultTimeout = 5 * time.Second

	// defaultQueueCap is the minimum and default capacity
	// of the async actions queue.
	defaultQueueCap = 256
)

//--------------------
// HELPER
//--------------------

// Action defines the signature of an actor action.
type Action func()

// Repairer allows the Actor to react on a panic during its
// work. If it returns nil the backend shall continue
// work. Otherwise the error is stored and the backend
// terminated.
type Repairer func(reason any) error

// Finalizer is called with the Actors internal status when
// the backend loop terminates.
type Finalizer func(err error) error

//--------------------
// ACTOR
//--------------------

// Actor allows to simply use and control a goroutine and sending
// functions to be executed sequentially by that goroutine.
type Actor struct {
	mu           sync.Mutex
	ctx          context.Context
	timeout      time.Duration
	cancel       func()
	asyncActions chan Action
	syncActions  chan Action
	repairer     Repairer
	finalizer    Finalizer
	works        atomic.Value
	err          error
}

// Go starts an Actor with the given options.
func Go(options ...Option) (*Actor, error) {
	// Init with options.
	act := &Actor{
		syncActions: make(chan Action),
	}
	act.works.Store(true)
	for _, option := range options {
		if err := option(act); err != nil {
			return nil, err
		}
	}
	// Ensure default settings.
	if act.ctx == nil {
		act.ctx, act.cancel = context.WithCancel(context.Background())
	} else {
		act.ctx, act.cancel = context.WithCancel(act.ctx)
	}
	if act.timeout == 0 {
		act.timeout = defaultTimeout
	}
	if act.asyncActions == nil {
		act.asyncActions = make(chan Action, defaultQueueCap)
	}
	// Create loop with its options.
	started := make(chan struct{})
	go act.backend(started)
	select {
	case <-started:
		return act, nil
	case <-time.After(act.timeout):
		return nil, fmt.Errorf("timeout starting actor after %.1f seconds", act.timeout.Seconds())
	}
}

// DoAsync sends the actor function to the backend goroutine and returns
// when it's queued.
func (act *Actor) DoAsync(action Action) error {
	return act.DoAsyncTimeout(action, act.timeout)
}

// DoAsyncTimeout send the actor function to the backend and returns
// when it's queued.
func (act *Actor) DoAsyncTimeout(action Action, timeout time.Duration) error {
	act.mu.Lock()
	if act.err != nil {
		act.mu.Unlock()
		return act.err
	}
	if !act.works.Load().(bool) {
		act.mu.Unlock()
		return fmt.Errorf("actor doesn't work anymore")
	}
	act.mu.Unlock()
	select {
	case act.asyncActions <- action:
	case <-time.After(timeout):
		return fmt.Errorf("timeout")
	}
	return nil
}

// DoSync executes the actor function and returns when it's done
// or it has the default timeout.
func (act *Actor) DoSync(action Action) error {
	return act.DoSyncTimeout(action, act.timeout)
}

// DoSyncTimeout executes the action and returns when it's done
// or it has a timeout.
func (act *Actor) DoSyncTimeout(action Action, timeout time.Duration) error {
	act.mu.Lock()
	if act.err != nil {
		act.mu.Unlock()
		return act.err
	}
	if !act.works.Load().(bool) {
		act.mu.Unlock()
		return fmt.Errorf("actor doesn't work anymore")
	}
	act.mu.Unlock()
	done := make(chan struct{})
	syncAction := func() {
		action()
		close(done)
	}
	select {
	case act.syncActions <- syncAction:
	case <-time.After(timeout):
		return fmt.Errorf("timeout")
	}
	select {
	case <-done:
	case <-time.After(timeout):
		if !act.works.Load().(bool) {
			return act.err
		}
		return fmt.Errorf("timeout")
	}
	return nil
}

// Err returns information if the Actor has an error.
func (act *Actor) Err() error {
	act.mu.Lock()
	defer act.mu.Unlock()
	return act.err
}

// Stop terminates the Actor backend.
func (act *Actor) Stop() {
	act.mu.Lock()
	defer act.mu.Unlock()
	if !act.works.Load().(bool) {
		// Already stopped.
		return
	}
	act.works.Store(false)
	act.cancel()
}

// backend runs the goroutine of the Actor.
func (act *Actor) backend(started chan struct{}) {
	defer act.finalize()
	close(started)
	for act.works.Load().(bool) {
		act.work()
	}
}

// work runs the select in a loop, including
// a possible repairer.
func (act *Actor) work() {
	defer func() {
		// Check and handle panics!
		reason := recover()
		switch {
		case reason != nil && act.repairer != nil:
			// Try to repair.
			err := act.repairer(reason)
			act.mu.Lock()
			act.err = err
			act.works.Store(act.err == nil)
			act.mu.Unlock()
		case reason != nil && act.repairer == nil:
			// Accept panic.
			act.mu.Lock()
			act.err = fmt.Errorf("actor panic: %v", reason)
			act.works.Store(false)
			act.mu.Unlock()
		}
	}()
	// Select in loop.
	for {
		select {
		case <-act.ctx.Done():
			return
		case action := <-act.asyncActions:
			action()
		case action := <-act.syncActions:
			action()
		}
	}
}

// finalize takes care for a clean loop finalization.
func (act *Actor) finalize() {
	act.mu.Lock()
	defer act.mu.Unlock()
	if act.finalizer != nil {
		act.err = act.finalizer(act.err)
	}
}

// EOF
