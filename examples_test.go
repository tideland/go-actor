// Tideland Go Actor - Unit Tests
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor_test

import (
	"context"
	"fmt"
	"time"

	"tideland.dev/go/actor"
)

// ExampleSimple demonstrates the basic usage of an actor.
func Example_simple() {
	// Create a default actor.
	act, err := actor.Go(actor.DefaultConfig())
	if err != nil {
		panic(err)
	}
	defer act.Stop()

	// Perform a synchronous action.
	err = act.DoSync(func() {
		fmt.Println("Hello, Actor!")
	})
	if err != nil {
		panic(err)
	}

	// Output:
	// Hello, Actor!
}

// ExampleWithFinalizer shows how to use a finalizer to clean up resources.
func Example_withFinalizer() {
	// Create a channel to signal finalization.
	finalized := make(chan struct{})

	// Configure the actor with a finalizer.
	cfg := actor.DefaultConfig()
	cfg.Finalizer = func(err error) error {
		fmt.Println("Finalizer: actor stopped")
		close(finalized)
		return err
	}

	// Create and start the actor.
	act, err := actor.Go(cfg)
	if err != nil {
		panic(err)
	}

	// Stop the actor and wait for the finalizer to complete.
	act.Stop()
	<-finalized

	// Output:
	// Finalizer: actor stopped
}

// ExampleWithRecoverer shows how to recover from panics within an actor.
func Example_withRecoverer() {
	// Create a channel to signal recovery.
	recovered := make(chan any, 1)

	// Configure the actor with a recoverer.
	cfg := actor.DefaultConfig()
	cfg.Recoverer = func(reason any) error {
		recovered <- reason
		return nil // Returning nil allows the actor to continue running.
	}

	// Create and start the actor.
	act, err := actor.Go(cfg)
	if err != nil {
		panic(err)
	}
	defer act.Stop()

	// Perform an action that will panic.
	act.DoSync(func() {
		panic("something went wrong")
	})

	// Wait for the recoverer to be called and print the reason.
	reason := <-recovered
	fmt.Printf("Recovered from panic: %v", reason)

	// Output:
	// Recovered from panic: something went wrong
}

// ExampleSyncAndAsync demonstrates synchronous and asynchronous actions.
func Example_syncAndAsync() {
	// Create a default actor.
	act, err := actor.Go(actor.DefaultConfig())
	if err != nil {
		panic(err)
	}
	defer act.Stop()

	// Synchronous action.
	act.DoSync(func() {
		fmt.Println("Sync: Hello from inside the actor!")
	})

	// Asynchronous action.
	done := make(chan struct{})
	act.DoAsync(func() {
		fmt.Println("Async: Hello from inside the actor!")
		close(done)
	})

	// Wait for the asynchronous action to complete.
	<-done

	// Output:
	// Sync: Hello from inside the actor!
	// Async: Hello from inside the actor!
}

// ExampleRepeat demonstrates repeating actions.
func Example_repeat() {
	// Create a default actor.
	act, err := actor.Go(actor.DefaultConfig())
	if err != nil {
		panic(err)
	}

	counter := 0
	stop, err := act.Repeat(10*time.Millisecond, func() {
		if counter < 5 {
			fmt.Println("Repeating...")
		}
		counter++
	})
	if err != nil {
		panic(err)
	}

	// Let the repeating action run for a while.
	time.Sleep(60 * time.Millisecond)

	// Stop the repeating action and the actor.
	stop()
	act.Stop()

	// Output:
	// Repeating...
	// Repeating...
	// Repeating...
	// Repeating...
	// Repeating...
}

// ExampleWithContext demonstrates using a context to control the actor's lifecycle.
func Example_withContext() {
	// Create a context that can be canceled.
	ctx, cancel := context.WithCancel(context.Background())

	// Configure the actor to use the context.
	cfg := actor.DefaultConfig()
	cfg.Context = ctx

	// Create and start the actor.
	act, err := actor.Go(cfg)
	if err != nil {
		panic(err)
	}

	// Cancel the context to stop the actor.
	cancel()

	// Check that the actor is done.
	if act.IsDone() {
		fmt.Println("Actor stopped via context cancellation.")
	}

	// Output:
	// Actor stopped via context cancellation.
}

// Example_struct demonstrates how to use an actor to protect the state of a
// struct. The Counter struct has an internal actor that serializes access to
// the value field. This ensures that all method calls are thread-safe.
func Example_struct() {
	// Counter is a simple struct that uses an actor to protect its state.
	type Counter struct {
		value int
		act   *actor.Actor
	}

	// NewCounter creates a new Counter.
	NewCounter := func() (*Counter, error) {
		c := &Counter{}
		act, err := actor.Go(actor.DefaultConfig())
		if err != nil {
			return nil, err
		}
		c.act = act
		return c, nil
	}

	// Increment increases the counter's value by one.
	Increment := func(c *Counter) {
		c.act.DoAsync(func() {
			c.value++
		})
	}

	// Value returns the current value of the counter.
	Value := func(c *Counter) int {
		var value int
		c.act.DoSync(func() {
			value = c.value
		})
		return value
	}

	// Stop stops the counter's actor.
	Stop := func(c *Counter) {
		c.act.Stop()
	}

	// Usage:
	counter, err := NewCounter()
	if err != nil {
		panic(err)
	}

	// Increment the counter ten times concurrently.
	for i := 0; i < 10; i++ {
		go Increment(counter)
	}

	// Wait for the increments to complete.
	// A sync call will block until all async calls are done.
	cvalue := 0
	for cvalue < 10 {
		time.Sleep(10 * time.Millisecond)
		cvalue = Value(counter)
	}

	fmt.Printf("Counter value: %d", cvalue)

	Stop(counter)

	// Output:
	// Counter value: 10
}
