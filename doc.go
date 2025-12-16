// Tideland Go Actor
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

/*
Package actor provides a robust and easy-to-use implementation of the Actor Model in Go.
It simplifies concurrent programming by allowing you to encapsulate state and behavior
within actors. All actions on an actor's state are executed sequentially in a
dedicated background goroutine, eliminating the need for manual locking and reducing
the risk of race conditions.

The package is designed to be flexible and extensible, with support for features like
synchronous and asynchronous actions, panic recovery, context-based cancellation,
and repeating tasks.

Usage:

To create an actor, use the `actor.Go()` function with a configuration. The default
configuration is often a good starting point:

	act, err := actor.Go(actor.DefaultConfig())
	if err != nil {
		// Handle error.
	}
	defer act.Stop()

Actions can be performed on the actor synchronously or asynchronously:

	// Synchronous action, blocks until complete.
	err := act.DoSync(func() {
		fmt.Println("This runs inside the actor.")
	})

	// Asynchronous action, returns immediately.
	err = act.DoAsync(func() {
		fmt.Println("This runs inside the actor, too.")
	})

Protecting Struct State:

A common use case for actors is to protect the state of a struct by embedding an
actor within it. This serializes access to the struct's fields and ensures that
all method calls are thread-safe.

	type Counter struct {
		value int
		act   *actor.Actor
	}

	func NewCounter() (*Counter, error) {
		c := &Counter{}
		act, err := actor.Go(actor.DefaultConfig())
		if err != nil {
			return nil, err
		}
		c.act = act
		return c, nil
	}

	func (c *Counter) Increment() {
		c.act.DoAsync(func() {
			c.value++
		})
	}

	func (c *Counter) Value() int {
		var value int
		c.act.DoSync(func() {
			value = c.value
		})
		return value
	}

Features:

  - Synchronous and Asynchronous Actions: Choose between `DoSync` for blocking
    operations and `DoAsync` for non-blocking operations. `DoSync` can also be
    used with a `context` for timeouts.

  - Panic Recovery: The `Recoverer` function in the configuration allows you to
    gracefully handle panics that occur within an actor's actions.

  - Finalization: The `Finalizer` function is called when the actor stops,
    allowing you to perform cleanup tasks.

  - Context Integration: Actors can be controlled using a `context.Context`,
    allowing them to be stopped when the context is canceled.

  - Repeating Actions: The `Repeat` method allows you to schedule a function to be
    called at a regular interval.

For more detailed examples, see the `examples_test.go` file.
*/

package actor
