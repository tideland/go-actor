// Tideland Go Actor - Unit Tests
//
// Copyright (C) 2019-2023 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor_test

//--------------------
// IMPORTS
//--------------------

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"tideland.dev/go/audit/asserts"

	"tideland.dev/go/actor"
)

//--------------------
// TESTS
//--------------------

// TestPureOK verifies the starting and stopping an Actor.
func TestPureOK(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	finalized := make(chan struct{})
	act, err := actor.Go(actor.WithFinalizer(func(err error) error {
		defer close(finalized)
		return err
	}))
	assert.OK(err)
	assert.NotNil(act)

	act.Stop()

	<-finalized

	assert.NoError(act.Err())
}

// TestPureDoubleStop verifies stopping an Actor twice.
func TestPureDoubleStop(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	finalized := make(chan struct{})
	act, err := actor.Go(actor.WithFinalizer(func(err error) error {
		defer close(finalized)
		return err
	}))
	assert.OK(err)
	assert.NotNil(act)

	act.Stop()
	act.Stop()

	<-finalized

	assert.NoError(act.Err())
}

// TestPureError verifies starting and stopping an Actor.
// Returning the stop error.
func TestPureError(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	finalized := make(chan struct{})
	act, err := actor.Go(actor.WithFinalizer(func(err error) error {
		defer close(finalized)
		return errors.New("damn")
	}))
	assert.OK(err)
	assert.NotNil(act)

	act.Stop()

	<-finalized

	assert.ErrorMatch(act.Err(), "damn")
}

// TestContext verifies starting and stopping an Actor
// with an external context.
func TestContext(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	ctx, cancel := context.WithCancel(context.Background())
	act, err := actor.Go(actor.WithContext(ctx))
	assert.OK(err)
	assert.NotNil(act)

	cancel()
	assert.NoError(act.Err())
}

// TestSync verifies synchronous calls.
func TestSync(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	finalized := make(chan struct{})
	act, err := actor.Go(actor.WithFinalizer(func(err error) error {
		defer close(finalized)
		return err
	}))
	assert.OK(err)

	counter := 0

	for i := 0; i < 5; i++ {
		assert.OK(act.DoSync(func() {
			counter++
		}))
	}

	assert.Equal(counter, 5)

	act.Stop()

	<-finalized

	assert.ErrorMatch(act.DoSync(func() {
		counter++
	}), "actor is done")
}

// TestTimeout verifies timout error of a synchronous Action.
func TestTimeout(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	act, err := actor.Go()
	assert.OK(err)

	// Scenario: Timeout is shorter than needed time, so error
	// is returned.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = act.DoSyncWithContext(ctx, func() {
		time.Sleep(25 * time.Millisecond)
	})
	assert.NoError(err)
	cancel()
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = act.DoSyncWithContext(ctx, func() {
		time.Sleep(100 * time.Millisecond)
	})
	assert.ErrorMatch(err, "action.*context deadline exceeded.*")
	cancel()

	time.Sleep(150 * time.Millisecond)
	act.Stop()
}

// TestWithTimeoutContext verifies timout error of a synchronous Action
// when the Actor is configured with a context timeout.
func TestWithTimeoutContext(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	act, err := actor.Go(actor.WithContext(ctx))
	assert.OK(err)

	// Scenario: Configured timeout is shorter than needed
	// time, so error is returned.
	err = act.DoSync(func() {
		time.Sleep(10 * time.Millisecond)
	})
	assert.NoError(err)
	err = act.DoSync(func() {
		time.Sleep(100 * time.Millisecond)
	})
	assert.ErrorMatch(err, "actor.*context deadline exceeded.*")

	act.Stop()
	cancel()
}

// TestAsyncWithQueueCap tests running multiple calls asynchronously.
func TestAsyncWithQueueCap(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	act, err := actor.Go(actor.WithQueueCap(128))
	assert.OK(err)

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
		assert.OK(act.DoAsync(func() {
			time.Sleep(2 * time.Millisecond)
			sigs <- struct{}{}
		}))
	}
	enqueued := time.Since(now)

	// Expect signal done to be sent about one second later.
	<-done
	duration := time.Since(now)

	assert.OK((duration - 250*time.Millisecond) > enqueued)

	act.Stop()
}

// TestRecovererOK tests successful handling of panic recoveries.
func TestNotifierOK(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	counter := 0
	recovered := false
	done := make(chan struct{})
	recoverer := func(reason any) error {
		defer close(done)
		recovered = true
		return nil
	}
	act, err := actor.Go(actor.WithRecoverer(recoverer))
	assert.OK(err)

	act.DoSync(func() {
		counter++
		// Will crash on first call.
		fmt.Printf("%v", counter/(counter-1))
	})
	<-done
	assert.True(recovered)
	err = act.DoSync(func() {
		counter++
	})
	assert.OK(err)
	assert.Equal(counter, 2)

	act.Stop()
}

// TestRecovererFail tests failing handling of panic recoveries.
func TestNotifierFail(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	counter := 0
	recovered := false
	done := make(chan struct{})
	recoverer := func(reason any) error {
		defer close(done)
		recovered = true
		return fmt.Errorf("ouch: %v", reason)
	}
	act, err := actor.Go(actor.WithRecoverer(recoverer))
	assert.OK(err)

	act.DoSync(func() {
		counter++
		// Will crash on first call.
		fmt.Printf("%v", counter/(counter-1))
	})
	<-done
	assert.True(recovered)

	assert.True(act.IsDone())
	assert.ErrorMatch(act.Err(), "ouch:.*")
}

// EOF
