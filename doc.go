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
//   - Context-based cancellation and timeout support
//   - Periodic tasks with configurable intervals
//   - Graceful shutdown with finalizer support
//   - Configurable action queue capacity
//
// Basic Usage:
//
//	type Counter struct {
//		value int
//		act   *actor.Actor
//	}
//
//	func NewCounter() (*Counter, error) {
//		cfg := actor.DefaultConfig()
//		act, err := actor.Go(cfg)
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
// Advanced Configuration:
//
//	cfg := actor.Config{
//		Context:   ctx,              // Custom context for lifetime control
//		QueueCap:  1000,            // Set queue capacity to 1000 actions
//		Recoverer: func(r any) error {
//			log.Printf("Recovered from: %v", r)
//			return nil  // Continue execution
//		},
//		Finalizer: func(err error) error {
//			// Cleanup when actor stops
//			if err != nil {
//				log.Printf("Actor stopped with: %v", err)
//			}
//			return err
//		},
//	}
//	act, err := actor.Go(cfg)
//
// The Config struct allows customizing:
//   - Context: Controls the actor's lifetime
//   - QueueCap: Size of the action queue (must be positive)
//   - Recoverer: Custom panic recovery function
//   - Finalizer: Cleanup function called when actor stops
//
// Default configuration values are provided by actor.DefaultConfig():
//   - Context: context.Background()
//   - QueueCap: 256
//   - Recoverer: Wraps panic value in error
//   - Finalizer: Returns error unchanged
//
// For more examples and detailed information, see the package documentation
// and examples in the repository.
package actor // import "tideland.dev/go/actor"

// EOF