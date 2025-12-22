// Tideland Go Actor - Examples
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"tideland.dev/go/actor"
)

// Example_simple demonstrates the basic usage of an actor with encapsulated state.
func Example_simple() {
	// Define state type
	type Counter struct {
		value int
	}

	// Create actor with initial state
	cfg := actor.NewConfig(context.Background())
	counter, err := actor.Go(Counter{value: 0}, cfg)
	if err != nil {
		panic(err)
	}
	defer counter.Stop()

	// Perform a synchronous action on the state
	err = counter.Do(func(s *Counter) {
		s.value++
		fmt.Printf("Counter value: %d\n", s.value)
	})
	if err != nil {
		panic(err)
	}

	// Output:
	// Counter value: 1
}

// Example_bankAccount shows how to use an actor to protect complex state.
func Example_bankAccount() {
	type Account struct {
		balance int
		name    string
	}

	cfg := actor.NewConfig(context.Background())
	account, err := actor.Go(Account{balance: 100, name: "Savings"}, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer account.Stop()

	// Deposit money
	_ = account.Do(func(s *Account) {
		s.balance += 50
	})

	// Withdraw with validation using Update
	withdrawn, err := account.Update(func(s *Account) (any, error) {
		if s.balance >= 30 {
			s.balance -= 30
			return true, nil
		}
		return false, fmt.Errorf("insufficient funds")
	})

	fmt.Printf("Withdrawn: %v, Error: %v\n", withdrawn, err)

	// Check balance
	balance, _ := account.Query(func(s *Account) any {
		return s.balance
	})

	fmt.Printf("Final balance: %d\n", balance)
	// Output:
	// Withdrawn: true, Error: <nil>
	// Final balance: 120
}

// Example_configuration demonstrates the fluent configuration builder pattern.
func Example_configuration() {
	ctx := context.Background()

	cfg := actor.NewConfig(ctx).
		SetQueueCapacity(512).
		SetActionTimeout(5 * time.Second).
		SetShutdownTimeout(10 * time.Second).
		SetFinalizer(func(err error) error {
			fmt.Println("Actor stopped")
			return nil
		})

	// Check for configuration errors
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	type State struct{}
	act, _ := actor.Go(State{}, cfg)

	fmt.Println("Actor configured and running")

	// Stop actor and wait for finalizer
	act.Stop()
	<-act.Done()

	// Output:
	// Actor configured and running
	// Actor stopped
}

// Example_withContext demonstrates using a context to control the actor's lifecycle.
func Example_withContext() {
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())

	type State struct{ running bool }
	cfg := actor.NewConfig(ctx)
	act, _ := actor.Go(State{running: true}, cfg)

	// Actor runs
	fmt.Printf("Running: %v\n", act.IsRunning())

	// Cancel context
	cancel()
	<-act.Done()

	// Actor stopped
	fmt.Printf("Running: %v\n", act.IsRunning())
	// Output:
	// Running: true
	// Running: false
}

// Example_finalizer shows how to use a finalizer to clean up resources.
func Example_finalizer() {
	type Resource struct {
		connections int
	}

	cfg := actor.NewConfig(context.Background()).
		SetFinalizer(func(err error) error {
			fmt.Printf("Cleanup called\n")
			return nil
		})

	resource, _ := actor.Go(Resource{connections: 5}, cfg)

	_ = resource.Do(func(s *Resource) {
		s.connections++
	})

	resource.Stop()
	<-resource.Done()

	// Output: Cleanup called
}

// Example_repeatingActions demonstrates repeating actions at intervals.
func Example_repeatingActions() {
	type Stats struct {
		checks int
	}

	cfg := actor.NewConfig(context.Background())
	stats, _ := actor.Go(Stats{}, cfg)
	defer stats.Stop()

	// Run health check every 50ms
	stop := stats.Repeat(50*time.Millisecond, func(s *Stats) {
		s.checks++
		if s.checks <= 3 {
			fmt.Printf("Check %d\n", s.checks)
		}
	})

	time.Sleep(200 * time.Millisecond)
	stop() // Stop repeating

	// Output:
	// Check 1
	// Check 2
	// Check 3
}

// Example_syncAndAsync demonstrates synchronous and asynchronous actions.
func Example_syncAndAsync() {
	type Counter struct {
		value int
	}

	cfg := actor.NewConfig(context.Background())
	counter, _ := actor.Go(Counter{value: 0}, cfg)
	defer counter.Stop()

	// Synchronous action (blocks until complete)
	_ = counter.Do(func(s *Counter) {
		s.value++
		fmt.Println("Sync: incremented")
	})

	// Asynchronous action (returns immediately)
	_ = counter.DoAsync(func(s *Counter) {
		s.value++
		fmt.Println("Async: incremented")
	})

	// Wait for async to complete
	time.Sleep(10 * time.Millisecond)

	// Get final value
	value, _ := counter.Query(func(s *Counter) any {
		return s.value
	})
	fmt.Printf("Final value: %d\n", value)

	// Output:
	// Sync: incremented
	// Async: incremented
	// Final value: 2
}

// Example_concurrentSafety demonstrates true encapsulation and concurrent safety.
func Example_concurrentSafety() {
	type Counter struct {
		value int
	}

	cfg := actor.NewConfig(context.Background())
	counter, _ := actor.Go(Counter{value: 0}, cfg)
	defer counter.Stop()

	// Increment the counter ten times concurrently
	for range 10 {
		go func() {
			_ = counter.DoAsync(func(s *Counter) {
				s.value++
			})
		}()
	}

	// Wait for all increments to complete
	time.Sleep(50 * time.Millisecond)

	// Get final value - guaranteed to be 10 due to serialization
	value, _ := counter.Query(func(s *Counter) any {
		return s.value
	})
	fmt.Printf("Counter value: %d\n", value)

	// Output:
	// Counter value: 10
}

// Example_timeout demonstrates action timeout handling.
func Example_timeout() {
	type Processor struct{}

	cfg := actor.NewConfig(context.Background()).
		SetActionTimeout(50 * time.Millisecond)

	proc, _ := actor.Go(Processor{}, cfg)
	defer proc.Stop()

	// This will timeout
	err := proc.Do(func(s *Processor) {
		time.Sleep(100 * time.Millisecond)
	})

	if err != nil {
		fmt.Println("Action timed out")
	}

	// Output:
	// Action timed out
}

// Example_errorHandling demonstrates error handling from actions.
func Example_errorHandling() {
	type Account struct {
		balance int
	}

	cfg := actor.NewConfig(context.Background())
	account, _ := actor.Go(Account{balance: 100}, cfg)
	defer account.Stop()

	// Try to withdraw more than balance
	err := account.DoWithError(func(s *Account) error {
		if s.balance < 200 {
			return fmt.Errorf("insufficient funds")
		}
		s.balance -= 200
		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Output:
	// Error: insufficient funds
}

// Example_asyncAwait demonstrates queueing work and waiting for it later.
func Example_asyncAwait() {
	type Processor struct {
		processed int
	}

	cfg := actor.NewConfig(context.Background())
	proc, _ := actor.Go(Processor{}, cfg)
	defer proc.Stop()

	// Queue multiple operations and collect awaiters
	var awaiters []func() error
	for range 3 {
		await := proc.DoAsyncAwait(func(s *Processor) {
			s.processed++
		})
		awaiters = append(awaiters, await)
	}

	fmt.Println("All operations queued")

	// Do other work here...

	// Now wait for all operations to complete
	for _, await := range awaiters {
		_ = await()
	}

	// Check result
	count, _ := proc.Query(func(s *Processor) any {
		return s.processed
	})
	fmt.Printf("Processed: %d\n", count)

	// Output:
	// All operations queued
	// Processed: 3
}
