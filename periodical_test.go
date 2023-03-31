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
	"testing"
	"time"

	"tideland.dev/go/actor"
	"tideland.dev/go/audit/asserts"
)

//--------------------
// TESTS
//--------------------

// TestPeriodicalStopActor verifies Periodical working and being
// stopped when the Actor is stopped.
func TestPeriodicalStopActor(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	finalized := make(chan struct{})
	counter := 0
	act, err := actor.Go(actor.WithFinalizer(func(err error) error {
		defer close(finalized)

		counter = 0

		return err
	}))
	assert.OK(err)
	assert.NotNil(act)

	// Start the periodical.
	stop, err := act.Periodical(10*time.Millisecond, func() {
		counter++
	})
	assert.OK(err)
	assert.NotNil(stop)

	time.Sleep(100 * time.Millisecond)
	assert.True(counter >= 9, "possibly only 9 due to late interval start")

	// Stop the Actor and check the finalization.
	act.Stop()

	<-finalized

	assert.NoError(act.Err())
	assert.Equal(counter, 0)

	// Check if the Interval is stopped too.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(counter, 0)
}

// TestPeriodicalStopInterval verifies Periodical working and being
// stopped when the periodical is stopped.
func TestIntervalStopInterval(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	counter := 0
	act, err := actor.Go()
	assert.OK(err)
	assert.NotNil(act)

	// Start the Interval.
	stop, err := act.Periodical(10*time.Millisecond, func() {
		counter++
	})
	assert.OK(err)
	assert.NotNil(stop)

	time.Sleep(100 * time.Millisecond)
	assert.True(counter >= 9, "possibly only 9 due to late interval start")

	// Stop the periodical and check that it doesn't work anymore.
	counterNow := counter
	stop()

	time.Sleep(100 * time.Millisecond)
	assert.Equal(counter, counterNow)

	act.Stop()
}

// EOF
