// Tideland Go Actor
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

// Package actor provides a robust implementation of the Actor model pattern for concurrent
// programming in Go. It enables safe and efficient concurrent operations by ensuring that
// all actions on shared state are executed sequentially in a dedicated background goroutine.
//
// Key Features:
//   - Sequential execution of actions in a background goroutine
//   - Support for both synchronous and asynchronous operations
//   - Built-in panic recovery and error handling
//   - Configurable queue capacity for pending actions
//   - Context-based cancellation and timeout support
//   - Optional repeating actions with specified intervals
//   - Clean shutdown with finalizer support
//
// Basic Usage:
//
//	type Counter struct {
//		value int
//		act   *actor.Actor
//	}
//
//	func NewCounter() (*Counter, error) {
//		act, err := actor.Go()
//		if err != nil {
//			return nil, err
//		}
//		return &Counter{act: act}, nil
//	}
//
//	// Asynchronous increment
//	func (c *Counter) Increment() error {
//		return c.act.DoAsync(func() {
//			c.value++
//		})
//	}
//
//	// Synchronous read
//	func (c *Counter) Value() (int, error) {
//		var v int
//		err := c.act.DoSync(func() {
//			v = c.value
//		})
//		return v, err
//	}
//
// Advanced Features:
//
// 1. Context Support:
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//	defer cancel()
//	
//	err := actor.DoSyncWithContext(ctx, func() {
//		// Long-running operation
//	})
//
// 2. Periodic Actions:
//
//	// Execute action every second
//	actor.Repeat(time.Second, func() {
//		// Periodic task
//	})
//
// 3. Custom Error Recovery:
//
//	act, err := actor.Go(
//		actor.WithRecoverer(func(reason any) error {
//			log.Printf("Recovered from panic: %v", reason)
//			return nil // Continue execution
//		}),
//	)
//
// 4. Graceful Shutdown:
//
//	act, _ := actor.Go(
//		actor.WithFinalizer(func(err error) error {
//			// Cleanup resources
//			return err
//		}),
//	)
//
// The actor package is particularly useful when building concurrent applications
// that need to maintain consistent state without explicit locking mechanisms.
// It helps prevent race conditions and makes concurrent code easier to reason about
// by centralizing state modifications in a single goroutine.
package actor // import "tideland.dev/go/actor"

// EOF