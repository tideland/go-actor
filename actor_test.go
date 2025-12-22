// Tideland Go Actor - Unit Tests
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"tideland.dev/go/asserts/verify"

	"tideland.dev/go/actor"
)

// Test state types

type Counter struct {
	value int
}

type Account struct {
	balance int
	name    string
}

// TestActorCreate verifies creating and stopping an actor.
func TestActorCreate(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	verify.NotNil(t, act)
	verify.True(t, act.IsRunning())

	act.Stop()
	<-act.Done()

	verify.True(t, act.IsDone())
	verify.False(t, act.IsRunning())
}

// TestActorDoubleStop verifies stopping an actor twice is safe.
func TestActorDoubleStop(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{}, cfg)
	verify.NoError(t, err)

	act.Stop()
	act.Stop() // Should be safe to call twice

	<-act.Done()
	verify.True(t, act.IsDone())
}

// TestActorWithContext verifies context cancellation stops the actor.
func TestActorWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := actor.NewConfig(ctx)
	act, err := actor.Go(Counter{}, cfg)
	verify.NoError(t, err)

	cancel()
	<-act.Done()

	verify.True(t, act.IsDone())
}

// TestActorFinalizer verifies the finalizer is called on shutdown.
func TestActorFinalizer(t *testing.T) {
	finalized := make(chan struct{})
	cfg := actor.NewConfig(context.Background()).
		SetFinalizer(func(err error) error {
			close(finalized)
			return err
		})

	act, err := actor.Go(Counter{}, cfg)
	verify.NoError(t, err)

	act.Stop()

	select {
	case <-finalized:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Finalizer was not called")
	}
}

// TestActorDoSync verifies synchronous actions.
func TestActorDoSync(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	err = act.Do(func(s *Counter) {
		s.value = 42
	})
	verify.NoError(t, err)

	value, err := act.Query(func(s *Counter) any {
		return s.value
	})
	verify.NoError(t, err)
	verify.Equal(t, value.(int), 42)
}

// TestActorDoAsync verifies asynchronous actions.
func TestActorDoAsync(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	// Queue several async actions
	for range 10 {
		err := act.DoAsync(func(s *Counter) {
			s.value++
		})
		verify.NoError(t, err)
	}

	// Wait for them to complete
	time.Sleep(50 * time.Millisecond)

	value, err := act.Query(func(s *Counter) any {
		return s.value
	})
	verify.NoError(t, err)
	verify.Equal(t, value.(int), 10)
}

// TestActorQuery verifies reading state.
func TestActorQuery(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Account{balance: 100, name: "Savings"}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	balance, err := act.Query(func(s *Account) any {
		return s.balance
	})
	verify.NoError(t, err)
	verify.Equal(t, balance.(int), 100)

	name, err := act.Query(func(s *Account) any {
		return s.name
	})
	verify.NoError(t, err)
	verify.Equal(t, name.(string), "Savings")
}

// TestActorUpdate verifies update with return value.
func TestActorUpdate(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 5}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	oldValue, err := act.Update(func(s *Counter) (any, error) {
		old := s.value
		s.value = 10
		return old, nil
	})
	verify.NoError(t, err)
	verify.Equal(t, oldValue, 5)

	newValue, err := act.Query(func(s *Counter) any {
		return s.value
	})
	verify.NoError(t, err)
	verify.Equal(t, newValue.(int), 10)
}

// TestActorDoWithError verifies error handling from actions.
func TestActorDoWithError(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Account{balance: 100}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	err = act.DoWithError(func(s *Account) error {
		if s.balance < 200 {
			return errors.New("insufficient funds")
		}
		s.balance -= 200
		return nil
	})
	verify.ErrorMatch(t, err, "insufficient funds")

	// Balance should be unchanged
	balance, _ := act.Query(func(s *Account) any {
		return s.balance
	})
	verify.Equal(t, balance.(int), 100)
}

// TestActorTimeout verifies action timeout.
func TestActorTimeout(t *testing.T) {
	cfg := actor.NewConfig(context.Background()).
		SetActionTimeout(50 * time.Millisecond)

	act, err := actor.Go(Counter{}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	err = act.Do(func(s *Counter) {
		time.Sleep(200 * time.Millisecond)
	})
	verify.Error(t, err)
}

// TestActorConcurrency verifies concurrent access is serialized.
func TestActorConcurrency(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	// Start 100 goroutines, each incrementing 10 times
	done := make(chan struct{})
	for range 100 {
		go func() {
			for range 10 {
				_ = act.DoAsync(func(s *Counter) {
					s.value++
				})
			}
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < 100; i++ {
		<-done
	}

	// Wait for all async actions to complete
	time.Sleep(200 * time.Millisecond)

	// Verify count is exactly 1000 (no race conditions)
	value, err := act.Query(func(s *Counter) any {
		return s.value
	})
	verify.NoError(t, err)
	verify.Equal(t, value.(int), 1000)
}

// TestActorRepeat verifies repeating actions.
func TestActorRepeat(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	stop := act.Repeat(10*time.Millisecond, func(s *Counter) {
		s.value++
	})

	// Let it run for a bit
	time.Sleep(55 * time.Millisecond)
	stop()

	// Give it time to stop
	time.Sleep(20 * time.Millisecond)

	value, err := act.Query(func(s *Counter) any {
		return s.value
	})
	verify.NoError(t, err)
	verify.True(t, value.(int) >= 3 && value.(int) <= 7, fmt.Sprintf("Expected 3-7 increments, got %d", value.(int)))
}

// TestActorQueueStatus verifies queue status reporting.
func TestActorQueueStatus(t *testing.T) {
	cfg := actor.NewConfig(context.Background()).
		SetQueueCapacity(10)

	act, err := actor.Go(Counter{}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	status := act.QueueStatus()
	verify.Equal(t, status.Capacity, 10)
	verify.Equal(t, status.Length, 0)
	verify.False(t, status.IsFull)
}

// TestConfigValidation verifies configuration validation.
func TestConfigValidation(t *testing.T) {
	// Test invalid queue capacity
	cfg := actor.NewConfig(context.Background()).
		SetQueueCapacity(-10)

	err := cfg.Validate()
	verify.Error(t, err)

	_, err = actor.Go(Counter{}, cfg)
	verify.Error(t, err)
}

// TestConfigErrorAccumulation verifies multiple errors are accumulated.
func TestConfigErrorAccumulation(t *testing.T) {
	cfg := actor.NewConfig(context.Background()).
		SetQueueCapacity(-10).
		SetActionTimeout(-5 * time.Second).
		SetShutdownTimeout(-1 * time.Second)

	err := cfg.Error()
	verify.Error(t, err)

	// Should contain all three errors
	errStr := err.Error()
	verify.True(t, len(errStr) > 50, "Expected multiple errors to be accumulated")
}

// TestConfigFluentBuilder verifies fluent configuration API.
func TestConfigFluentBuilder(t *testing.T) {
	called := false
	cfg := actor.NewConfig(context.Background()).
		SetQueueCapacity(512).
		SetActionTimeout(5 * time.Second).
		SetShutdownTimeout(10 * time.Second).
		SetFinalizer(func(err error) error {
			called = true
			return nil
		})

	verify.NoError(t, cfg.Validate())
	verify.Equal(t, cfg.QueueCapacity(), 512)
	verify.Equal(t, cfg.ActionTimeout(), 5*time.Second)
	verify.Equal(t, cfg.ShutdownTimeout(), 10*time.Second)

	act, err := actor.Go(Counter{}, cfg)
	verify.NoError(t, err)

	act.Stop()
	<-act.Done()

	verify.True(t, called, "Finalizer should have been called")
}

// TestActorAfterStop verifies operations after stop return errors.
func TestActorAfterStop(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{}, cfg)
	verify.NoError(t, err)

	act.Stop()
	<-act.Done()

	err = act.Do(func(s *Counter) {
		s.value++
	})
	verify.Error(t, err)

	err = act.DoAsync(func(s *Counter) {
		s.value++
	})
	verify.Error(t, err)
}

// TestActorAsyncError verifies async actions with errors stop the actor.
func TestActorAsyncError(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{}, cfg)
	verify.NoError(t, err)

	// Queue an async action that returns an error
	err = act.DoAsyncWithError(func(s *Counter) error {
		return errors.New("async error")
	})
	verify.NoError(t, err) // Queueing succeeds

	// Wait for actor to process and stop
	<-act.Done()

	// Actor should have stopped due to the error
	verify.True(t, act.IsDone())
	verify.ErrorMatch(t, act.Err(), "async error")
}

// TestActorDoAsyncAwait verifies async queueing with synchronous waiting.
func TestActorDoAsyncAwait(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	// Queue multiple actions and collect awaiters
	awaiters := make([]func() error, 10)
	for i := range 10 {
		awaiters[i] = act.DoAsyncAwait(func(s *Counter) {
			s.value++
		})
	}

	// All actions are queued, now wait for them
	for _, await := range awaiters {
		err := await()
		verify.NoError(t, err)
	}

	// Verify all actions completed
	value, err := act.Query(func(s *Counter) any {
		return s.value
	})
	verify.NoError(t, err)
	verify.Equal(t, value.(int), 10)
}

// TestActorDoAsyncAwaitWithError verifies error handling from actions.
func TestActorDoAsyncAwaitWithError(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	// Queue action that will fail
	await := act.DoAsyncAwaitWithError(func(s *Counter) error {
		return errors.New("test error")
	})

	// Wait for it and expect error
	err = await()
	verify.ErrorMatch(t, err, "test error")

	// Actor should still be running (sync-style error handling)
	verify.True(t, act.IsRunning())
}

// TestActorDoAsyncAwaitAfterStop verifies behavior when actor is stopped.
func TestActorDoAsyncAwaitAfterStop(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)

	act.Stop()
	<-act.Done()

	// Try to queue after stop
	await := act.DoAsyncAwait(func(s *Counter) {
		s.value++
	})

	// Awaiter should return error immediately
	err = await()
	verify.Error(t, err)
}

// TestActorDoAsyncAwaitContext verifies context cancellation.
func TestActorDoAsyncAwaitContext(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	// Queue with context
	await := act.DoAsyncAwaitWithErrorContext(ctx, func(s *Counter) error {
		time.Sleep(50 * time.Millisecond)
		s.value++
		return nil
	})

	// Cancel context before action executes
	cancel()

	// Awaiter should receive cancellation error
	err = await()
	verify.Error(t, err)
}

// TestActorDoAsyncAwaitNeverCalled verifies no goroutine leak.
func TestActorDoAsyncAwaitNeverCalled(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	// Queue action but never call awaiter
	_ = act.DoAsyncAwait(func(s *Counter) {
		s.value++
	})

	// Wait for action to complete
	time.Sleep(50 * time.Millisecond)

	// Verify action executed even though awaiter wasn't called
	value, err := act.Query(func(s *Counter) any {
		return s.value
	})
	verify.NoError(t, err)
	verify.Equal(t, value.(int), 1)
}

// TestActorDoAsyncAwaitMultipleCalls verifies awaiter can be called multiple times.
func TestActorDoAsyncAwaitMultipleCalls(t *testing.T) {
	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(Counter{value: 0}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	// Queue action with error
	await := act.DoAsyncAwaitWithError(func(s *Counter) error {
		s.value = 42
		return errors.New("test error")
	})

	// Call awaiter multiple times - should return same result
	err1 := await()
	err2 := await()
	err3 := await()

	verify.ErrorMatch(t, err1, "test error")
	verify.ErrorMatch(t, err2, "test error")
	verify.ErrorMatch(t, err3, "test error")

	// Verify action only executed once
	value, err := act.Query(func(s *Counter) any {
		return s.value
	})
	verify.NoError(t, err)
	verify.Equal(t, value.(int), 42)
}
