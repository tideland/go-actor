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
	"math/rand"
	"testing"
	"time"

	"tideland.dev/go/audit/asserts"

	"tideland.dev/go/actor"
)

//--------------------
// TESTS
//--------------------

// TestMass verifies the starting and stopping an Actor.
func TestMass(t *testing.T) {
	assert := asserts.NewTesting(t, asserts.FailStop)
	pps := make([]*PingPong, 1000)
	for i := 0; i < len(pps); i++ {
		pps[i] = NewPingPong(pps)
	}
	// Let's start the ping pong party.
	for i := 0; i < 5; i++ {
		n := rand.Intn(len(pps))
		pps[n].Ping()
		n = rand.Intn(len(pps))
		pps[n].Pong()
	}
	// Let's wat 5 seconds before stopping.
	time.Sleep(5 * time.Second)
	// Let's stop the actors.
	for _, pp := range pps {
		pings, pongs := pp.PingPongs()
		assert.True(pings > 0)
		assert.True(pongs > 0)
		pp.Stop()
		assert.NoError(pp.Err())
	}
}

//--------------------
// TEST ACTOR
//--------------------

type PingPong struct {
	pps   []*PingPong
	pings int
	pongs int

	act *actor.Actor
}

func NewPingPong(pps []*PingPong) *PingPong {
	pp := &PingPong{
		pps:   pps,
		pings: 0,
		pongs: 0,
	}
	act, err := actor.Go()
	if err != nil {
		panic(err)
	}
	pp.act = act
	return pp
}

func (pp *PingPong) Ping() {
	pp.act.DoAsync(func() {
		pp.pings++
		n := rand.Intn(len(pp.pps))
		pp.pps[n].Pong()
	})
}

func (pp *PingPong) Pong() {
	pp.act.DoAsync(func() {
		pp.pongs++
		n := rand.Intn(len(pp.pps))
		pp.pps[n].Ping()
	})
}

func (pp *PingPong) PingPongs() (int, int) {
	var pings int
	var pongs int
	pp.act.DoSync(func() {
		pings = pp.pings
		pongs = pp.pongs
	})
	return pings, pongs
}

func (pp *PingPong) Err() error {
	return pp.act.Err()
}

func (pp *PingPong) Stop() {
	pp.act.Stop()
}

// EOF
