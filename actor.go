package actor

// Actor - Encapsulates state of type S and ensures all access is serialized.
// Actor OWNS the state, making race conditions impossible by design.
//
// This follows the Erlang/OTP process model where:
// - The actor encapsulates state (like an Erlang process)
// - State is only accessible through message passing (closures)
// - All state modifications are serialized automatically
//
// Panics in actions will crash the actor's goroutine (as they should in Go).

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Finalizer is a function called when an actor stops.
// It receives the error that caused the shutdown (if any).
// Returning an error replaces the shutdown error.
type Finalizer func(err error) error

// QueueStatus contains information about the actor's request queue.
type QueueStatus struct {
	Length   int  // Current number of queued requests
	Capacity int  // Maximum queue capacity
	IsFull   bool // Whether the queue is at capacity
}

// Actor encapsulates state of type S and processes actions sequentially.
type Actor[S any] struct {
	state    S
	requests chan *request[S]
	ctx      context.Context
	cancel   func()
	err      atomic.Pointer[error]
	status   atomic.Bool
	done     chan struct{}
	config   *Config
}

// request represents a queued action to be performed on the state
type request[S any] struct {
	ctx    context.Context
	action func(*S) error
	done   chan error
}

// Go starts a new actor with the given initial state and configuration.
// The state is fully encapsulated and can only be accessed through Do methods.
//
// Example:
//
//	type Counter struct { value int }
//	actor, err := actor.Go(Counter{}, actor.NewConfig(ctx))
func Go[S any](initialState S, cfg *Config) (*Actor[S], error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(cfg.Context())

	a := &Actor[S]{
		state:    initialState,
		requests: make(chan *request[S], cfg.QueueCapacity()),
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
		config:   cfg,
	}

	go a.run()
	return a, nil
}

// run is the main goroutine that processes all state modifications sequentially.
// Panics in actions will crash this goroutine and stop the actor.
func (a *Actor[S]) run() {
	defer close(a.done)
	defer a.status.Store(true)

	var finalErr error

	// Call finalizer when we exit (if configured)
	defer func() {
		if finalizer := a.config.Finalizer(); finalizer != nil {
			if err := finalizer(finalErr); err != nil {
				finalErr = err
			}
		}
		a.err.Store(&finalErr)
	}()

	for {
		select {
		case <-a.ctx.Done():
			finalErr = &ActorError{
				Op:   "run",
				Err:  a.ctx.Err(),
				Code: ErrShutdown,
			}
			return

		case req := <-a.requests:
			err := a.executeRequest(req)
			if err != nil {
				// Error from action execution, stop actor
				finalErr = err
				return
			}
		}
	}
}

// executeRequest runs a single request.
// Returns error only if the action fails and it's an async action
// (which causes the actor to stop).
func (a *Actor[S]) executeRequest(req *request[S]) error {
	// Check if request context is done
	select {
	case <-req.ctx.Done():
		err := &ActorError{
			Op:   "execute",
			Err:  req.ctx.Err(),
			Code: ErrCanceled,
		}
		if req.done != nil {
			req.done <- err
		}
		return nil
	default:
	}

	var actionErr error

	// Apply action timeout if configured
	if timeout := a.config.ActionTimeout(); timeout > 0 {
		ctx, cancel := context.WithTimeout(req.ctx, timeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			actionErr = req.action(&a.state)
			close(done)
		}()

		select {
		case <-done:
			// Action completed
		case <-ctx.Done():
			actionErr = &ActorError{
				Op:   "execute",
				Err:  ctx.Err(),
				Code: ErrTimeout,
			}
		}
	} else {
		// No timeout, execute directly
		actionErr = req.action(&a.state)
	}

	// Send result back if synchronous
	if req.done != nil {
		req.done <- actionErr
		return nil
	}

	// Async action - if it failed, stop the actor
	return actionErr
}

// Do executes an action synchronously on the encapsulated state.
// The action receives a pointer to the state and can modify it.
// This blocks until the action completes.
//
// Example:
//
//	err := actor.Do(func(s *Counter) {
//	    s.value++
//	})
func (a *Actor[S]) Do(action func(*S)) error {
	return a.DoWithError(func(s *S) error {
		action(s)
		return nil
	})
}

// DoWithError executes an action synchronously that can return an error.
func (a *Actor[S]) DoWithError(action func(*S) error) error {
	return a.DoWithErrorContext(a.ctx, action)
}

// DoWithErrorContext executes an action synchronously with a custom context.
func (a *Actor[S]) DoWithErrorContext(ctx context.Context, action func(*S) error) error {
	if a.IsDone() {
		return &ActorError{
			Op:   "do",
			Err:  a.Err(),
			Code: ErrShutdown,
		}
	}

	req := &request[S]{
		ctx:    ctx,
		action: action,
		done:   make(chan error, 1),
	}

	select {
	case a.requests <- req:
		return <-req.done
	case <-ctx.Done():
		return &ActorError{
			Op:   "do",
			Err:  ctx.Err(),
			Code: ErrCanceled,
		}
	case <-a.ctx.Done():
		return &ActorError{
			Op:   "do",
			Err:  a.ctx.Err(),
			Code: ErrShutdown,
		}
	}
}

// DoWithErrorTimeout executes an action with a timeout.
func (a *Actor[S]) DoWithErrorTimeout(timeout time.Duration, action func(*S) error) error {
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	defer cancel()
	return a.DoWithErrorContext(ctx, action)
}

// DoAsync executes an action asynchronously on the state.
// Returns immediately without waiting for the action to complete.
// If the action returns an error, the actor will stop.
//
// Example:
//
//	err := actor.DoAsync(func(s *Counter) {
//	    s.value++
//	})
func (a *Actor[S]) DoAsync(action func(*S)) error {
	return a.DoAsyncWithError(func(s *S) error {
		action(s)
		return nil
	})
}

// DoAsyncWithError executes an action asynchronously that can return an error.
// Errors from async actions will cause the actor to stop.
func (a *Actor[S]) DoAsyncWithError(action func(*S) error) error {
	return a.DoAsyncWithErrorContext(a.ctx, action)
}

// DoAsyncWithErrorContext executes an action asynchronously with a custom context.
func (a *Actor[S]) DoAsyncWithErrorContext(ctx context.Context, action func(*S) error) error {
	if a.IsDone() {
		return &ActorError{
			Op:   "do-async",
			Err:  a.Err(),
			Code: ErrShutdown,
		}
	}

	req := &request[S]{
		ctx:    ctx,
		action: action,
		done:   nil, // No response channel = async
	}

	select {
	case a.requests <- req:
		return nil
	case <-ctx.Done():
		return &ActorError{
			Op:   "do-async",
			Err:  ctx.Err(),
			Code: ErrCanceled,
		}
	case <-a.ctx.Done():
		return &ActorError{
			Op:   "do-async",
			Err:  a.ctx.Err(),
			Code: ErrShutdown,
		}
	}
}

// DoAsyncAwait queues an action asynchronously and returns an awaiter function.
// The awaiter blocks until the action completes and returns the action's error.
// The awaiter function is safe to call multiple times - it will return the same
// result each time.
//
// This is useful when you want to queue work immediately but wait for it later:
//
//	await := actor.DoAsyncAwait(func(s *Counter) {
//	    s.value++
//	})
//	// Do other work...
//	err := await() // Now wait for the action to complete
func (a *Actor[S]) DoAsyncAwait(action func(*S)) func() error {
	return a.DoAsyncAwaitWithError(func(s *S) error {
		action(s)
		return nil
	})
}

// DoAsyncAwaitWithError queues an action that can return an error and returns an awaiter.
// The awaiter function blocks until the action completes and returns the action's error.
// The awaiter function is safe to call multiple times - it will return the same result each time.
func (a *Actor[S]) DoAsyncAwaitWithError(action func(*S) error) func() error {
	return a.DoAsyncAwaitWithErrorContext(a.ctx, action)
}

// DoAsyncAwaitWithErrorContext queues an action with a custom context and returns an awaiter.
// The awaiter function blocks until the action completes and returns the action's error.
// The awaiter function is safe to call multiple times - it will return the same result each time.
func (a *Actor[S]) DoAsyncAwaitWithErrorContext(ctx context.Context, action func(*S) error) func() error {
	// Create done channel immediately for result delivery
	done := make(chan error, 1)
	var queueErr error

	// Try to queue the request
	if a.IsDone() {
		queueErr = &ActorError{
			Op:   "do-async-await",
			Err:  a.Err(),
			Code: ErrShutdown,
		}
	} else {
		req := &request[S]{
			ctx:    ctx,
			action: action,
			done:   done, // Has done channel = result will be sent back
		}

		select {
		case a.requests <- req:
			// Successfully queued
		case <-ctx.Done():
			queueErr = &ActorError{
				Op:   "do-async-await",
				Err:  ctx.Err(),
				Code: ErrCanceled,
			}
		case <-a.ctx.Done():
			queueErr = &ActorError{
				Op:   "do-async-await",
				Err:  a.ctx.Err(),
				Code: ErrShutdown,
			}
		}
	}

	// Return awaiter function that caches the result using sync.Once
	var once sync.Once
	var result error

	return func() error {
		once.Do(func() {
			if queueErr != nil {
				result = queueErr
			} else {
				result = <-done
			}
		})
		return result
	}
}

// Query retrieves a value from the state synchronously.
// This is a convenience method for read-only operations.
//
// Example:
//
//	value, err := actor.Query(func(s *Counter) int {
//	    return s.value
//	})
func (a *Actor[S]) Query(getter func(*S) any) (any, error) {
	var result any
	err := a.Do(func(s *S) {
		result = getter(s)
	})
	return result, err
}

// Update modifies the state and returns a result in a single atomic operation.
// This is useful when you need to both modify state and return something.
//
// Example:
//
//	oldValue, err := actor.Update(func(s *Counter) (int, error) {
//	    old := s.value
//	    s.value++
//	    return old, nil
//	})
func (a *Actor[S]) Update(updater func(*S) (any, error)) (any, error) {
	var result any
	err := a.DoWithError(func(s *S) error {
		var err error
		result, err = updater(s)
		return err
	})
	return result, err
}

// Stop gracefully shuts down the actor.
func (a *Actor[S]) Stop() {
	a.cancel()
}

// Done returns a channel that is closed when the actor stops.
func (a *Actor[S]) Done() <-chan struct{} {
	return a.done
}

// IsDone returns true if the actor has stopped.
func (a *Actor[S]) IsDone() bool {
	return a.status.Load()
}

// IsRunning returns true if the actor is still running.
func (a *Actor[S]) IsRunning() bool {
	return !a.IsDone()
}

// Err returns any error that caused the actor to stop.
func (a *Actor[S]) Err() error {
	if err := a.err.Load(); err != nil {
		return *err
	}
	return nil
}

// QueueStatus returns information about the request queue.
func (a *Actor[S]) QueueStatus() QueueStatus {
	length := len(a.requests)
	capacity := cap(a.requests)
	return QueueStatus{
		Length:   length,
		Capacity: capacity,
		IsFull:   length == capacity,
	}
}

// Repeat executes an action at regular intervals until stopped.
// Returns a function that stops the repetition.
//
// Example:
//
//	stop := actor.Repeat(1*time.Second, func(s *Counter) {
//	    log.Printf("Counter value: %d", s.value)
//	})
//	defer stop()
func (a *Actor[S]) Repeat(interval time.Duration, action func(*S)) func() {
	return a.RepeatWithContext(a.ctx, interval, action)
}

// RepeatWithContext executes an action at intervals with cancellation support.
func (a *Actor[S]) RepeatWithContext(ctx context.Context, interval time.Duration, action func(*S)) func() {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-a.ctx.Done():
				return
			case <-ticker.C:
				_ = a.DoAsyncWithErrorContext(ctx, func(s *S) error {
					action(s)
					return nil
				})
			}
		}
	}()

	return cancel
}
