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

// request wraps an action with its context.
type request struct {
	ctx    context.Context
	done   chan struct{}
	err    error
	action Action
}

// newRequest creates a request including a done channel. The
// Action is wrapped with a closure which closes the done channel
// after the action has been executed.
func newRequest(ctx context.Context, action Action) *request {
	return &request{
		ctx:    ctx,
		done:   make(chan struct{}),
		action: action,
	}
}

// execute checks if the request context is canceled or timed out.
// If not, it performs the action and closes the done channel.
func (req *request) execute() error {
	defer close(req.done)
	d, ok := req.ctx.Deadline()
	if ok {
		select {
		case <-req.ctx.Done():
			return req.ctx.Err()
		case <-time.After(time.Until(d)):
			return fmt.Errorf("actor request timed out")
		default:
		}
	}
	req.action()
	return nil
}

// Actor introduces the actor model, where call simply are executed
// sequentially in a backend goroutine.
type Actor struct {
	ctx           context.Context
	cancel        func()
	asyncRequests chan *request
	syncRequests  chan *request
	recoverer     Recoverer
	finalizer     Finalizer
	err           atomic.Pointer[error]
	done          chan struct{}
}

// Go starts an Actor with the given options.
func Go(options ...Option) (*Actor, error) {
	// Init with options.
	act := &Actor{
		ctx:          context.Background(),
		syncRequests: make(chan *request),
	}
	for _, option := range options {
		if err := option(act); err != nil {
			return nil, err
		}
	}
	// Ensure default settings.
	act.ctx, act.cancel = context.WithCancel(act.ctx)
	if act.asyncRequests == nil {
		act.asyncRequests = make(chan *request, defaultQueueCap)
	}
	if act.recoverer == nil {
		act.recoverer = func(reason any) error {
			return fmt.Errorf("panic during actor action: %v", reason)
		}
	}
	if act.finalizer == nil {
		act.finalizer = func(err error) error { return err }
	}
	// Start the backend, wait for it to be ready.
	started := make(chan struct{})

	go act.backend(started)

	select {
	case <-started:
	case <-time.After(time.Second):
		return nil, fmt.Errorf("actor backend did not start")
	}
	return act, nil
}

// DoAsync sends the actor function to the backend goroutine and returns
// when it's queued.
func (act *Actor) DoAsync(action Action) error {
	return act.DoAsyncWithContext(context.Background(), action)
}

// DoAsyncWithContext send the actor function to the backend and returns
// when it's queued. A context allows to cancel the action or add a timeout.
func (act *Actor) DoAsyncWithContext(ctx context.Context, action Action) error {
	// Check if we're error free and still working.
	if act.err.Load() != nil {
		return *act.err.Load()
	}
	if act.IsDone() {
		return fmt.Errorf("actor is done")
	}
	// Send action to backend.
	req := newRequest(ctx, action)
	select {
	case act.asyncRequests <- req:
	case <-req.ctx.Done():
		return fmt.Errorf("action timed out or cancelled")
	case <-act.ctx.Done():
		return fmt.Errorf("actor timed out or cancelled")
	}
	return nil
}

// DoSync executes the actor function and returns when it's done.
func (act *Actor) DoSync(action Action) error {
	return act.DoSyncWithContext(context.Background(), action)
}

// DoSyncWithContext executes the action and returns when it's done.
// A context allows to cancel the action or add a timeout.
func (act *Actor) DoSyncWithContext(ctx context.Context, action Action) error {
	// Check if we're error free and still working.
	if act.err.Load() != nil {
		return *act.err.Load()
	}
	if act.IsDone() {
		return fmt.Errorf("actor is done")
	}
	// Send action to backend.
	req := newRequest(ctx, action)
	select {
	case act.syncRequests <- req:
	case <-req.ctx.Done():
		return fmt.Errorf("action timed out or cancelled")
	case <-ctx.Done():
		return fmt.Errorf("actor timed out or cancelled")
	}
	select {
	case <-req.done:
	case <-req.ctx.Done():
		return fmt.Errorf("action timed out or cancelled")
	case <-act.ctx.Done():
		return fmt.Errorf("actor timed out or cancelled")
	}
	return req.err
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
func (act *Actor) backend(started chan struct{}) {
	defer act.finalize()
	close(started)

	act.done = make(chan struct{})

	// Work as long as we're not stopped.
	for !act.IsDone() {
		act.work()
	}
}

// work runs the select in a loop, including
// a possible repairer.
func (act *Actor) work() {
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
		case <-act.ctx.Done():
			close(act.done)
			return
		case req := <-act.asyncRequests:
			req.err = req.execute()
		case req := <-act.syncRequests:
			req.err = req.execute()
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
