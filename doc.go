// Tideland Go Actor
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

/*
Package actor provides a robust implementation of the Actor Model in Go using generics
to encapsulate state. It simplifies concurrent programming by ensuring all access to
shared state is serialized, eliminating the need for manual locking and preventing
race conditions by design.

Following the Erlang/OTP process model, actors truly encapsulate state - the state is
owned by the actor and can only be accessed through message passing (closures). This
makes race conditions impossible since there's no way to bypass the actor's serialization.

Key Design Principles:

- State Encapsulation: The actor OWNS the state of generic type S. State cannot be
accessed directly, only through closures that receive *S.

- Sequential Execution: All actions on the state execute sequentially in a dedicated
goroutine, guaranteeing no race conditions.

- Type Safety: Using Go generics ensures type-safe state access without reflection.

- Configuration via Builder: Worker-style fluent configuration with error accumulation.

- No Panic Recovery: Panics crash the actor's goroutine as they should in Go, rather
than trying to continue with potentially corrupt state.

Basic Usage:

To create an actor, define your state type and use actor.Go() with a configuration:

	type Counter struct {
		value int
	}

	cfg := actor.NewConfig(context.Background())
	counter, err := actor.Go(Counter{value: 0}, cfg)
	if err != nil {
		// Handle error
	}
	defer counter.Stop()

Actions on State:

Actions are closures that receive a pointer to the state and can modify it:

	// Synchronous action (blocks until complete)
	err := counter.Do(func(s *Counter) {
		s.value++
	})

	// Asynchronous action (returns immediately)
	err = counter.DoAsync(func(s *Counter) {
		s.value++
	})

	// Query state (read-only pattern)
	value, err := counter.Query(func(s *Counter) int {
		return s.value
	})

	// Update and return a result atomically
	oldValue, err := counter.Update(func(s *Counter) (any, error) {
		old := s.value
		s.value = 10
		return old, nil
	})

Configuration:

The fluent configuration builder allows customization with error accumulation:

	cfg := actor.NewConfig(ctx).
		SetQueueCapacity(512).                    // Request queue size
		SetActionTimeout(5 * time.Second).        // Max action duration
		SetShutdownTimeout(10 * time.Second).     // Max shutdown wait
		SetFinalizer(func(err error) error {      // Cleanup on stop
			log.Printf("Actor stopped: %v", err)
			return nil
		})

	// Check for configuration errors
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	actor, err := actor.Go(MyState{}, cfg)

Features:

- Synchronous Actions: Do(), DoWithError(), DoWithErrorContext() block until complete

- Asynchronous Actions: DoAsync(), DoAsyncWithError() queue work and return immediately

- Queries: Query() for read-only access with type-safe return values

- Updates: Update() for atomic read-modify-write operations

- Repeating Actions: Repeat() schedules periodic execution

- Context Integration: Actors respect context cancellation for lifecycle management

- Timeout Support: Per-action timeouts and context-based cancellation

- Queue Status: QueueStatus() reports queue depth and capacity

- Error Handling: Actions can return errors; async errors stop the actor

Example - Bank Account:

	type Account struct {
		balance int
		name    string
	}

	cfg := actor.NewConfig(context.Background())
	account, _ := actor.Go(Account{balance: 100, name: "Savings"}, cfg)
	defer account.Stop()

	// Deposit
	account.Do(func(s *Account) {
		s.balance += 50
	})

	// Withdraw with validation
	withdrawn, err := account.Update(func(s *Account) (any, error) {
		if s.balance >= 30 {
			s.balance -= 30
			return true, nil
		}
		return false, fmt.Errorf("insufficient funds")
	})

	// Check balance
	balance, _ := account.Query(func(s *Account) int {
		return s.balance
	})

Why This Design?

This generic actor pattern solves a common problem with the embedding pattern: with
embedding, developers could accidentally write direct getters/setters that bypass the
actor, creating race conditions. By making the actor OWN the state, such bypasses
become impossible - the compiler prevents direct state access.

For more examples, see the examples_test.go file.
*/

package actor
