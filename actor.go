// Tideland Go Actor
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor

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
func (req *request) execute() {
	defer close(req.done)
	select {
	case <-req.ctx.Done():
		req.err = req.ctx.Err()
	default:
		req.action()
	}
}

// Actor introduces the actor model, where calls are executed
// sequentially in a backend goroutine.
// QueueStatus provides information about the actor's request queue
type QueueStatus struct {
	Length   int  // Current number of queued requests
	Capacity int  // Maximum queue capacity
	IsFull   bool // Whether queue is at capacity
}

type Actor struct {
	ctx       context.Context
	cancel    func()
	requests  chan *request
	recoverer Recoverer
	finalizer Finalizer
	err       atomic.Pointer[error]
	done      chan struct{}
	timeout   time.Duration // default timeout for actions from config
	status    atomic.Bool
}

// Go starts an Actor with the given configuration.
func Go(cfg Config) (*Actor, error) {
	// Validate configuration.
	if err := cfg.Validate(); err != nil {
		return nil, NewError("Go", err, ErrInvalid)
	}

	// Create actor with validated config.
	act := &Actor{
		requests:  make(chan *request, cfg.QueueCap),
		recoverer: cfg.Recoverer,
		finalizer: cfg.Finalizer,
		timeout:   cfg.ActionTimeout,
	}

	// Set up context with cancellation.
	act.ctx, act.cancel = context.WithCancel(cfg.Context)

	// Start the backend.
	started := make(chan struct{})
	go act.backend(started)

	// Wait for backend to start.
	select {
	case <-started:
		return act, nil
	case <-time.After(time.Second):
		return nil, NewError("Go", fmt.Errorf("backend did not start"), ErrTimeout)
	}
}

// DoAsync sends the actor function to the backend goroutine and returns
// when it's queued.
func (act *Actor) DoAsync(action Action) error {
	return act.DoAsyncWithContext(context.Background(), action)
}

// DoAsyncWithContext sends the actor function to the backend and returns
// when it's queued. A context allows to cancel the action or add a timeout.
func (act *Actor) DoAsyncWithContext(ctx context.Context, action Action) error {
	req := newRequest(ctx, action)
	return act.send(req)
}

// DoSync executes the actor function and returns when it's done.
func (act *Actor) DoSync(action Action) error {
	return act.DoSyncWithContext(context.Background(), action)
}

// DoSyncWithContext executes the action and returns when it's done.
// A context allows to cancel the action or add a timeout.
func (act *Actor) DoSyncWithContext(ctx context.Context, action Action) error {
	req := newRequest(ctx, action)
	err := act.send(req)
	if err != nil {
		return err
	}
	return act.wait(req)
}

// Done returns a channel that is closed when the Actor terminates.
func (act *Actor) Done() <-chan struct{} {
	return act.done
}

// IsDone allows to simply check if the Actor is done in a select
// or if statement.
func (act *Actor) IsDone() bool {
	return act.status.Load()
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

// QueueStatus returns the current status of the action queue
func (act *Actor) QueueStatus() QueueStatus {
	return QueueStatus{
		Length:   len(act.requests),
		Capacity: cap(act.requests),
		IsFull:   len(act.requests) == cap(act.requests),
	}
}

// DoSyncTimeout executes the action with a specific timeout
func (act *Actor) DoSyncTimeout(timeout time.Duration, action Action) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return act.DoSyncWithContext(ctx, action)
}

// DoAsyncTimeout sends the action to the backend with a specific timeout
func (act *Actor) DoAsyncTimeout(timeout time.Duration, action Action) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return act.DoAsyncWithContext(ctx, action)
}

// send sends a request to the backend.
func (act *Actor) send(req *request) error {
	// Check if we're error free and still working.
	if act.err.Load() != nil {
		return *act.err.Load()
	}
	if act.IsDone() {
		return fmt.Errorf("actor is done")
	}
	// Apply action timeout if configured
	if act.timeout > 0 && req.ctx == context.Background() {
		var cancel context.CancelFunc
		req.ctx, cancel = context.WithTimeout(req.ctx, act.timeout)
		defer cancel()
	}
	// Send the request to the backend.
	select {
	case act.requests <- req:
	case <-req.ctx.Done():
		return fmt.Errorf("action context sending: %v", req.ctx.Err())
	case <-act.ctx.Done():
		return fmt.Errorf("actor context sending: %v", act.ctx.Err())
	}
	return nil
}

// wait waits for synchronous requests to be done or returning an error.
func (act *Actor) wait(req *request) error {
	select {
	case <-req.done:
	case <-req.ctx.Done():
		return NewError("wait", fmt.Errorf("action context: %v", req.ctx.Err()), ErrCanceled)
	case <-act.ctx.Done():
		return NewError("wait", fmt.Errorf("actor context: %v", act.ctx.Err()), ErrShutdown)
	}
	if req.err != nil {
		return NewError("wait", req.err, ErrCanceled)
	}
	return nil
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
				act.status.Store(true)
				close(act.done)
			}
		}
	}()
	// Select in loop.
	for {
		select {
		case <-act.ctx.Done():
			act.status.Store(true)
			close(act.done)
			return
		case req := <-act.requests:
			req.execute()
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
