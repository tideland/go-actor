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

// TestPureOK verifies the starting and stopping an Actor.
func TestPureOK(t *testing.T) {
	finalized := make(chan struct{})
	cfg := actor.DefaultConfig()
	cfg.Finalizer = func(err error) error {
		defer close(finalized)
		return err
	}
	act, err := actor.Go(cfg)
	verify.NoError(t, err)
	verify.NotNil(t, act)

	act.Stop()

	<-finalized

	verify.NoError(t, act.Err())
}

// TestPureDoubleStop verifies stopping an Actor twice.
func TestPureDoubleStop(t *testing.T) {
	finalized := make(chan struct{})
	cfg := actor.DefaultConfig()
	cfg.Finalizer = func(err error) error {
		defer close(finalized)
		return err
	}
	act, err := actor.Go(cfg)
	verify.NoError(t, err)
	verify.NotNil(t, act)

	act.Stop()
	act.Stop()

	<-finalized

	verify.NoError(t, act.Err())
}

// TestPureError verifies starting and stopping an Actor.
// Returning the stop error.
func TestPureError(t *testing.T) {
	finalized := make(chan struct{})
	cfg := actor.DefaultConfig()
	cfg.Finalizer = func(err error) error {
		defer close(finalized)
		return errors.New("damn")
	}
	act, err := actor.Go(cfg)
	verify.NoError(t, err)
	verify.NotNil(t, act)

	act.Stop()

	<-finalized

	verify.ErrorMatch(t, act.Err(), "damn")
}

// TestContext verifies starting and stopping an Actor
// with an external context.
func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := actor.DefaultConfig()
	cfg.Context = ctx
	act, err := actor.Go(cfg)
	verify.NoError(t, err)
	verify.NotNil(t, act)

	cancel()
	verify.NoError(t, act.Err())
}

// TestSync verifies synchronous calls.
func TestSync(t *testing.T) {
	finalized := make(chan struct{})
	cfg := actor.DefaultConfig()
	cfg.Finalizer = func(err error) error {
		defer close(finalized)
		return err
	}
	act, err := actor.Go(cfg)
	verify.NoError(t, err)

	counter := 0

	for range 5 {
		verify.NoError(t, act.DoSync(func() {
			counter++
		}))
	}

	verify.Equal(t, counter, 5)

	act.Stop()

	<-finalized

	verify.ErrorMatch(t, act.DoSync(func() {
		counter++
	}), "actor is done")
}

// TestTimeout verifies timout error of a synchronous Action.
func TestTimeout(t *testing.T) {
	act, err := actor.Go(actor.DefaultConfig())
	verify.NoError(t, err)

	// Scenario: Timeout is shorter than needed time, so error
	// is returned.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = act.DoSyncWithContext(ctx, func() {
		time.Sleep(25 * time.Millisecond)
	})
	verify.NoError(t, err)
	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = act.DoSyncWithContext(ctx, func() {
		time.Sleep(100 * time.Millisecond)
	})
	verify.ErrorMatch(t, err, "action.*context deadline exceeded.*")
	cancel()

	time.Sleep(150 * time.Millisecond)
	act.Stop()
}

// TestWithTimeoutContext verifies timout error of a synchronous Action
// when the Actor is configured with a context timeout.
func TestWithTimeoutContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	cfg := actor.DefaultConfig()
	cfg.Context = ctx
	act, err := actor.Go(cfg)
	verify.NoError(t, err)

	// Scenario: Configured timeout is shorter than needed
	// time, so error is returned.
	err = act.DoSync(func() {
		time.Sleep(10 * time.Millisecond)
	})
	verify.NoError(t, err)
	err = act.DoSync(func() {
		time.Sleep(100 * time.Millisecond)
	})
	verify.ErrorMatch(t, err, "actor.*context deadline exceeded.*")

	act.Stop()
	cancel()
}

// TestAsyncWithQueueCap tests running multiple calls asynchronously.
func TestAsyncWithQueueCap(t *testing.T) {
	cfg := actor.DefaultConfig()
	cfg.QueueCap = 128
	act, err := actor.Go(cfg)
	verify.NoError(t, err)

	sigs := make(chan struct{}, 1)
	done := make(chan struct{}, 1)

	// Start background func waiting for the signals of
	// the asynchrounous calls.
	go func() {
		count := 0
		for range sigs {
			count++
			if count == 128 {
				break
			}
		}
		close(done)
	}()

	// Now start asynchrounous calls.
	now := time.Now()
	for range 128 {
		verify.NoError(t, act.DoAsync(func() {
			time.Sleep(2 * time.Millisecond)
			sigs <- struct{}{}
		}))
	}
	enqueued := time.Since(now)

	// Expect signal done to be sent about one second later.
	<-done
	duration := time.Since(now)

	verify.True(t, (duration-250*time.Millisecond) > enqueued)

	act.Stop()
}

// TestRecovererOK tests successful handling of panic recoveries.
func TestRecovererOK(t *testing.T) {
	counter := 0
	recovered := false
	done := make(chan struct{})
	cfg := actor.DefaultConfig()
	cfg.Recoverer = func(reason any) error {
		defer close(done)
		recovered = true
		return nil
	}
	act, err := actor.Go(cfg)
	verify.NoError(t, err)

	act.DoSync(func() {
		counter++
		// Will crash on first call.
		fmt.Printf("%v", counter/(counter-1))
	})
	<-done
	verify.True(t, recovered)
	err = act.DoSync(func() {
		counter++
	})
	verify.NoError(t, err)
	verify.Equal(t, counter, 2)

	act.Stop()
}

// TestRecovererFail tests failing handling of panic recoveries.
func TestRecovererFail(t *testing.T) {
	counter := 0
	recovered := false
	done := make(chan struct{})
	cfg := actor.DefaultConfig()
	cfg.Recoverer = func(reason any) error {
		defer close(done)
		recovered = true
		return fmt.Errorf("ouch: %v", reason)
	}
	act, err := actor.Go(cfg)
	verify.NoError(t, err)

	act.DoSync(func() {
		counter++
		// Will crash on first call.
		fmt.Printf("%v", counter/(counter-1))
	})
	<-done
	verify.True(t, recovered)

	verify.True(t, act.IsDone())
	verify.ErrorMatch(t, act.Err(), "ouch:.*")
}

// TestConcurrentAccess tests concurrent access to an actor.
func TestConcurrentAccess(t *testing.T) {
	act, err := actor.Go(actor.DefaultConfig())
	verify.NoError(t, err)
	defer act.Stop()

	const goroutines = 10
	const actionsPerGoroutine = 100

	counter := 0

	start := make(chan struct{})
	done := make(chan struct{})

	for range goroutines {
		go func() {
			<-start
			for range actionsPerGoroutine {
				act.DoSync(func() {
					counter++
				})
			}
			done <- struct{}{}
		}()
	}

	close(start)

	for range goroutines {
		<-done
	}

	verify.Equal(t, counter, goroutines*actionsPerGoroutine)
}

// TestIsRunning verifies the IsRunning() method.
func TestIsRunning(t *testing.T) {
	act, err := actor.Go(actor.DefaultConfig())
	verify.NoError(t, err)

	verify.True(t, act.IsRunning())

	act.Stop()
	<-act.Done()

	verify.False(t, act.IsRunning())
}

// BenchmarkGo benchmarks the creation of an actor.
func BenchmarkGo(b *testing.B) {
	for b.Loop() {
		act, err := actor.Go(actor.DefaultConfig())
		if err != nil {
			b.Fatalf("cannot create actor: %v", err)
		}
		act.Stop()
	}
}

// BenchmarkDoSync benchmarks synchronous actions.
func BenchmarkDoSync(b *testing.B) {
	act, err := actor.Go(actor.DefaultConfig())
	if err != nil {
		b.Fatalf("cannot create actor: %v", err)
	}
	defer act.Stop()

	for b.Loop() {
		act.DoSync(func() {
			// Noop.
		})
	}
}

// BenchmarkDoAsync benchmarks asynchronous actions.
func BenchmarkDoAsync(b *testing.B) {
	act, err := actor.Go(actor.DefaultConfig())
	if err != nil {
		b.Fatalf("cannot create actor: %v", err)
	}
	defer act.Stop()

	for b.Loop() {
		act.DoAsync(func() {
			// Noop.
		})
	}
}

// FuzzAction fuzzes actor actions.
func FuzzAction(f *testing.F) {
	f.Add("sync", 10)
	f.Add("async", 100)
	f.Add("sync", 0)

	f.Fuzz(func(t *testing.T, actionType string, numActions int) {
		act, err := actor.Go(actor.DefaultConfig())
		verify.NoError(t, err)

		counter := 0
		for range numActions {
			switch actionType {
			case "sync":
				act.DoSync(func() {
					counter++
				})
			case "async":
				act.DoAsync(func() {
					counter++
				})
			default:
				// Invalid action, just continue.
			}
		}

		// Use a sync action to wait for all async actions to complete.
		act.DoSync(func() {
			if numActions > 0 && (actionType == "sync" || actionType == "async") {
				verify.Equal(t, counter, numActions)
			} else {
				verify.Equal(t, counter, 0)
			}
		})

		act.Stop()
	})
}
