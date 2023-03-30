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

// Recoverer defines the signature of a function for recovering
// from a panic during executing an action. The reason is the
// panic value. The function should return the error to be
// returned by the Actor. If the error is nil, the Actor will
// continue to work.
type Recoverer func(reason any) error

// Finalizer defines the signature of a function for finalizing
// the work of an Actor. The error is the one returned by the
// Actor.
type Finalizer func(err error) error

//--------------------
// ACTOR
//--------------------

// Actor allows to simply use and control a goroutine and sending
// functions to be executed sequentially by that goroutine.
type Actor struct {
	timeout      time.Duration
	cancel       func()
	asyncActions chan Action
	syncActions  chan Action
	recoverer    Recoverer
	finalizer    Finalizer
	err          atomic.Pointer[error]
	done         chan struct{}
}

// GoContext starts an Actor with a context and the given options.
func GoContext(ctx context.Context, options ...Option) (*Actor, error) {
	// Init with options.
	act := &Actor{
		syncActions: make(chan Action),
		done:        make(chan struct{}),
	}
	for _, option := range options {
		if err := option(act); err != nil {
			return nil, err
		}
	}
	// Ensure default settings.
	ctx, act.cancel = context.WithCancel(ctx)
	if act.timeout == 0 {
		act.timeout = defaultTimeout
	}
	if act.asyncActions == nil {
		act.asyncActions = make(chan Action, defaultQueueCap)
	}
	if act.recoverer == nil {
		act.recoverer = func(reason any) error {
			return fmt.Errorf("panic during actor action: %v", reason)
		}
	}
	if act.finalizer == nil {
		act.finalizer = func(err error) error { return err }
	}
	// Create loop with its options.
	started := make(chan struct{})
	go act.backend(ctx, started)
	select {
	case <-started:
		return act, nil
	case <-time.After(act.timeout):
		return nil, fmt.Errorf("timeout starting actor after %.1f seconds", act.timeout.Seconds())
	}
}

// Go starts an Actor with the given options.
func Go(options ...Option) (*Actor, error) {
	return GoContext(context.Background(), options...)
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
	if act.err.Load() != nil {
		return *act.err.Load()
	}
	if act.IsDone() {
		return fmt.Errorf("actor is done")
	}
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
	if act.err.Load() != nil {
		return *act.err.Load()
	}
	if act.IsDone() {
		return fmt.Errorf("actor is done")
	}
	// Create signal channel for done work and send action. Then
	// wait for the signal or the timeout.
	actionDone := make(chan struct{})
	syncAction := func() {
		defer close(actionDone)
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
	case <-actionDone:
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for action execution")
	}
	return nil
}

// Done returns a channel that is closed when the Actor terminates.
func (act *Actor) Done() <-chan struct{} {
	return act.done
}

func (act *Actor) IsDone() bool {
	select {
	case <-act.done:
		return true
	default:
		return false
	}
}

// Err returns information if the Actor has an error.
func (act *Actor) Err() error {
	err := act.err.Load()
	if err == nil {
		return nil
	}
	return *err
}

// Stop terminates the Actor backend.
func (act *Actor) Stop() {
	if act.IsDone() {
		return
	}
	act.cancel()
}

// backend runs the goroutine of the Actor.
func (act *Actor) backend(ctx context.Context, started chan struct{}) {
	defer act.finalize()
	close(started)
	// Work as long as we're not stopped.
	for !act.IsDone() {
		act.work(ctx)
	}
}

// work runs the select in a loop, including
// a possible repairer.
func (act *Actor) work(ctx context.Context) {
	defer func() {
		// Check panics and possibly send notification.
		if reason := recover(); reason != nil {
			err := act.recoverer(reason)
			if err != nil {
				act.err.Store(&err)
				close(act.done)
			}
		}
	}()
	// Select in loop.
	for {
		select {
		case <-ctx.Done():
			close(act.done)
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
	var ferr error
	err := act.err.Load()
	if err != nil {
		ferr = act.finalizer(*err)
	} else {
		ferr = act.finalizer(nil)
	}
	if ferr != nil {
		act.err.Store(&ferr)
	}
}

// EOF
