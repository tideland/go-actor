// Tideland Go Actor - Mass Tests
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"tideland.dev/go/asserts/verify"

	"tideland.dev/go/actor"
)

// TestMass verifies large scale concurrent actor usage with ping-pong pattern.
func TestMass(t *testing.T) {
	// Create 1000 ping-pong actors
	pps := make([]*PingPongActor, 1000)
	for i := range pps {
		pps[i] = NewPingPongActor(pps)
	}

	// Start the ping pong party
	for range 5 {
		n := rand.Intn(len(pps))
		pps[n].Ping()
		n = rand.Intn(len(pps))
		pps[n].Pong()
	}

	// Let it run for a second
	time.Sleep(1 * time.Second)

	// Check some random ping pong pairs
	for _, pp := range pps {
		pings, pongs := pp.PingPongs()
		verify.True(t, pings > 0)
		verify.True(t, pongs > 0)
		pp.Stop()
	}
}

// TestPerformance verifies actor performance with many async operations.
func TestPerformance(t *testing.T) {
	type State struct{}

	cfg := actor.NewConfig(context.Background())
	act, err := actor.Go(State{}, cfg)
	verify.NoError(t, err)
	verify.NotNil(t, act)

	now := time.Now()
	for range 10000 {
		act.DoAsync(func(s *State) {})
	}
	duration := time.Since(now)
	verify.True(t, duration < 100*time.Millisecond, "Queueing 10000 operations took too long")

	act.Stop()
	<-act.Done()

	// Actor stopped normally - will have a shutdown error
	verify.Error(t, act.Err())
}

// PingPongState holds the state for a ping-pong actor.
type PingPongState struct {
	pings int
	pongs int
}

// PingPongActor wraps an actor with ping-pong convenience methods.
type PingPongActor struct {
	act *actor.Actor[PingPongState]
	pps []*PingPongActor
}

// NewPingPongActor creates a new ping-pong actor.
func NewPingPongActor(pps []*PingPongActor) *PingPongActor {
	cfg := actor.NewConfig(context.Background()).
		SetQueueCapacity(256)

	act, err := actor.Go(PingPongState{}, cfg)
	if err != nil {
		panic(err)
	}

	return &PingPongActor{
		act: act,
		pps: pps,
	}
}

// Ping increments pings and triggers a random Pong.
func (pp *PingPongActor) Ping() {
	pp.act.DoAsync(func(s *PingPongState) {
		s.pings++
		n := rand.Intn(len(pp.pps))
		pp.pps[n].Pong()
	})
}

// Pong increments pongs and triggers a random Ping.
func (pp *PingPongActor) Pong() {
	pp.act.DoAsync(func(s *PingPongState) {
		s.pongs++
		n := rand.Intn(len(pp.pps))
		pp.pps[n].Ping()
	})
}

// PingPongs returns the current ping and pong counts.
func (pp *PingPongActor) PingPongs() (int, int) {
	var pings, pongs int
	pp.act.Do(func(s *PingPongState) {
		pings = s.pings
		pongs = s.pongs
	})
	return pings, pongs
}

// Err returns any error from the actor.
func (pp *PingPongActor) Err() error {
	return pp.act.Err()
}

// Stop stops the actor.
func (pp *PingPongActor) Stop() {
	pp.act.Stop()
}
