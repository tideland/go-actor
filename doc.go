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
//   - Context-based cancellation support
//   - Action timeouts (global and per-action)
//   - Queue monitoring capabilities
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
//	// Synchronous read with timeout
//	func (c *Counter) Value() (int, error) {
//		var v int
//		err := c.act.DoSyncTimeout(time.Second, func() {
//			v = c.value
//		})
//		return v, err
//	}
//
// Configuration:
//
//	cfg := actor.Config{
//		Context:       ctx,               // Custom context
//		QueueCap:      1000,             // Queue capacity
//		ActionTimeout: 5 * time.Second,   // Default timeout
//		Recoverer: func(r any) error {
//			log.Printf("Panic: %v", r)
//			return nil
//		},
//		Finalizer: func(err error) error {
//			if err != nil {
//				log.Printf("Stopped with: %v", err)
//			}
//			return err
//		},
//	}
//
// Queue Monitoring:
//
//	status := act.QueueStatus()
//	if status.IsFull {
//		log.Printf("Queue at capacity: %d/%d", status.Length, status.Capacity)
//	}
//
// Timeout Handling:
//
//	// Global timeout via config
//	cfg := actor.DefaultConfig()
//	cfg.ActionTimeout = 5 * time.Second
//	act, _ := actor.Go(cfg)
//
//	// Per-action timeout
//	err := act.DoSyncTimeout(time.Second, func() {
//		// Operation with 1s timeout
//	})
//
//	// Context timeout
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//	defer cancel()
//	err = act.DoSyncWithContext(ctx, func() {
//		// Operation with context timeout
//	})
//
// The actor package is particularly useful when building concurrent applications
// that need to maintain consistent state without explicit locking mechanisms.
// It helps prevent race conditions and makes concurrent code easier to reason about
// by centralizing state modifications in a single goroutine.
package actor // import "tideland.dev/go/actor"

// EOF