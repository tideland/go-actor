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

// TestWithContext verifies starting and stopping an Actor
// with a context.
func TestWithContext(t *testing.T) {
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
	}), "actor doesn't work anymore")
}

// TestTimeout verifies timout error of a synchronous Action.
func TestTimeout(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	act, err := actor.Go()
	assert.OK(err)

	// Scenario: Timeout is shorter than needed time, so error
	// is returned.
	err = act.DoSyncTimeout(func() {
		time.Sleep(5 * time.Second)
	}, 500*time.Millisecond)

	assert.ErrorMatch(err, ".*timeout waiting.*")

	act.Stop()
}

// TestWithTimeout verifies timout error of a synchronous Action
// after setting it as option.
func TestWithTimeout(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	act, err := actor.Go(actor.WithTimeout(time.Second))
	assert.OK(err)

	// Scenario: Configured timeout is shorter than needed
	// time, so error is returned.
	err = act.DoSync(func() {
		time.Sleep(2 * time.Second)
	})

	assert.ErrorMatch(err, ".*timeout waiting.*")

	act.Stop()
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
	start := time.Now()
	for i := 0; i < 128; i++ {
		assert.OK(act.DoAsync(func() {
			time.Sleep(5 * time.Millisecond)
			sigs <- struct{}{}
		}))
	}
	enqueued := time.Since(start)

	// Expect signal done to be sent about one second later.
	<-done
	duration := time.Since(start)

	assert.OK((duration - 640*time.Millisecond) > enqueued)

	act.Stop()
}

// TestNotifierOK tests handling panic notifications successfully.
func TestNotifierOK(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	counter := 0
	notified := false
	done := make(chan struct{})
	notifier := func(reason any) {
		defer close(done)
		notified = true
	}
	act, err := actor.Go(actor.WithNotifier(notifier))
	assert.OK(err)

	act.DoSyncTimeout(func() {
		counter++
		// Will crash on first call.
		fmt.Printf("%v", counter/(counter-1))
	}, time.Second)
	<-done
	assert.True(notified)
	err = act.DoSync(func() {
		counter++
	})
	assert.OK(err)
	assert.Equal(counter, 2)

	act.Stop()
}

// EOF
