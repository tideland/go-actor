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

// Notifier allows the Actor notify an external entity
// about an internal panic when executing an action. It
// is just a notification with the reason. To change or
// fix data of the call use another action.
type Notifier func(reason any)

// Finalizer is called with the Actors internal error
// status when the Actor terminates.
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
	notifier     Notifier
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
	// Check if we're error free and still working.
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
	// Send action to backend.
	select {
	case act.asyncActions <- action:
	case <-time.After(timeout):
		return fmt.Errorf("timeout sending action")
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
	// Check if we're error free and still working.
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
	// Create signal channel for done work and send action. Then
	// wait for the signal or the timeout.
	done := make(chan struct{})
	syncAction := func() {
		defer close(done)
		action()
	}
	now := time.Now()
	select {
	case act.syncActions <- syncAction:
	case <-time.After(timeout):
		return fmt.Errorf("timeout sending action")
	}
	sent := time.Now()
	timeout = timeout - sent.Sub(now)
	select {
	case <-done:
	case <-time.After(timeout):
		if !act.works.Load().(bool) {
			return act.err
		}
		return fmt.Errorf("timeout waiting for done action")
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
	// Work as long as we're not stopped.
	for act.works.Load().(bool) {
		act.work()
	}
}

// work runs the select in a loop, including
// a possible repairer.
func (act *Actor) work() {
	defer func() {
		// Check panics and possibly send notification.
		if reason := recover(); reason != nil {
			if act.notifier != nil {
				go act.notifier(reason)
			}
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
