// Tideland Go Actor - Repeat Tests
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor_test

import (
	"context"
	"testing"
	"time"

	"tideland.dev/go/actor"
	"tideland.dev/go/asserts/verify"
)

// TestRepeatStopActor verifies Repeat working and being
// stopped when the Actor is stopped.
func TestRepeatStopActor(t *testing.T) {
	type State struct {
		counter int
	}

	finalized := make(chan struct{})
	cfg := actor.NewConfig(context.Background()).
		SetFinalizer(func(err error) error {
			close(finalized)
			return err
		})

	act, err := actor.Go(State{counter: 0}, cfg)
	verify.NoError(t, err)
	verify.NotNil(t, act)

	// Start the repeated action
	stop := act.Repeat(10*time.Millisecond, func(s *State) {
		s.counter++
	})
	verify.NotNil(t, stop)

	time.Sleep(100 * time.Millisecond)

	// Check counter value
	counter, _ := act.Query(func(s *State) any {
		return s.counter
	})
	verify.True(t, counter.(int) >= 9, "possibly only 9 due to late interval start")

	// Stop the Actor and check the finalization
	act.Stop()

	<-finalized

	// Actor stopped normally - will have a shutdown error
	verify.Error(t, act.Err())

	// Check if the repeat is stopped too
	time.Sleep(100 * time.Millisecond)
	// Can't query after stop, but the repeat should have stopped
}

// TestRepeatStopInterval verifies Repeat working and being
// stopped when the repeat is stopped.
func TestRepeatStopInterval(t *testing.T) {
	type State struct {
		counter int
	}

	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(State{counter: 0}, cfg)
	verify.NoError(t, err)
	verify.NotNil(t, act)
	defer act.Stop()

	// Start the repeated action
	stop := act.Repeat(10*time.Millisecond, func(s *State) {
		s.counter++
	})
	verify.NotNil(t, stop)

	time.Sleep(100 * time.Millisecond)

	// Get current counter value
	counterNow, _ := act.Query(func(s *State) any {
		return s.counter
	})
	verify.True(t, counterNow.(int) >= 9, "possibly only 9 due to late interval start")

	// Stop the repeat and check that it doesn't work anymore
	stop()

	time.Sleep(100 * time.Millisecond)

	// Counter should not have increased
	counterAfter, _ := act.Query(func(s *State) any {
		return s.counter
	})
	verify.Equal(t, counterAfter, counterNow)
}

// TestRepeatMultiple verifies multiple repeats can run concurrently.
func TestRepeatMultiple(t *testing.T) {
	type State struct {
		counter1 int
		counter2 int
	}

	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(State{}, cfg)
	verify.NoError(t, err)
	defer act.Stop()

	// Start two repeats with different intervals
	stop1 := act.Repeat(10*time.Millisecond, func(s *State) {
		s.counter1++
	})
	stop2 := act.Repeat(20*time.Millisecond, func(s *State) {
		s.counter2++
	})

	time.Sleep(100 * time.Millisecond)

	// Check both counters increased
	counter1, _ := act.Query(func(s *State) any {
		return s.counter1
	})
	counter2, _ := act.Query(func(s *State) any {
		return s.counter2
	})

	verify.True(t, counter1.(int) >= 9)
	verify.True(t, counter2.(int) >= 4)

	// Stop both
	stop1()
	stop2()
}
