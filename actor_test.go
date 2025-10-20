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

	for i := 0; i < 5; i++ {
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
	for i := 0; i < 128; i++ {
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
