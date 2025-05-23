// Tideland Go Actor - Unit Tests
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor_test

//--------------------
// IMPORTS
//--------------------

import (
	"testing"
	"time"

	"tideland.dev/go/actor"
	"tideland.dev/go/asserts/verify"
)

//--------------------
// TESTS
//--------------------

// TestRepeatStopActor verifies Repeat working and being
// stopped when the Actor is stopped.
func TestRepeatStopActor(t *testing.T) {
	finalized := make(chan struct{})
	counter := 0
	act, err := actor.Go(actor.WithFinalizer(func(err error) error {
		defer close(finalized)

		counter = 0

		return err
	}))
	verify.NoError(t, err)
	verify.NotNil(t, act)

	// Start the repeated action.
	stop, err := act.Repeat(10*time.Millisecond, func() {
		counter++
	})
	verify.NoError(t, err)
	verify.NotNil(t, stop)

	time.Sleep(100 * time.Millisecond)
	verify.True(t, counter >= 9, "possibly only 9 due to late interval start")

	// Stop the Actor and check the finalization.
	act.Stop()

	<-finalized

	verify.NoError(t, act.Err())
	verify.Equal(t, counter, 0)

	// Check if the Interval is stopped too.
	time.Sleep(100 * time.Millisecond)
	verify.Equal(t, counter, 0)
}

// TestRepeatStopInterval verifies Repeat working and being
// stopped when the repeat is stopped.
func TestRepeatStopInterval(t *testing.T) {
	counter := 0
	act, err := actor.Go()
	verify.NoError(t, err)
	verify.NotNil(t, act)

	// Start the repeated action.
	stop, err := act.Repeat(10*time.Millisecond, func() {
		counter++
	})
	verify.NoError(t, err)
	verify.NotNil(t, stop)

	time.Sleep(100 * time.Millisecond)
	verify.True(t, counter >= 9, "possibly only 9 due to late interval start")

	// Stop the repeat and check that it doesn't work anymore.
	counterNow := counter
	stop()

	time.Sleep(100 * time.Millisecond)
	verify.Equal(t, counter, counterNow)

	act.Stop()
}

// EOF
