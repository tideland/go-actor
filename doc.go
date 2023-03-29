// Tideland Go Actor
//
// Copyright (C) 2019-2023 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

// Package actor supports the simple creation of concurrent applications
// following the idea of actor models. The work to be done has to be defined
// as func() inside your public methods or functions and sent to the actor
// running in the background.
//
//		type Counter struct {
//			counter int
//			act     *actor.Actor
//		}
//
//		func NewCounter() (*Counter, error) {
//			act, err := actor.Go()
//			if err != nil {
//				return nil, err
//			}
//			c := &Counter{
//				counter: 0,
//				act:     act,
//			}
//	        c.startAutoIncrement()
//			return c, nil
//		}
//
//		func (c *Counter) Incr() error {
//			return c.act.DoAsync(func() {
//				c.counter++
//			})
//		}
//
//		func (c *Counter) Get() (int, error) {
//			var counter int
//			if err := c.act.DoSync(func() {
//				counter = c.counter
//			}); err != nil {
//				return 0, err
//			}
//			return counter, nil
//		}
//
//		func (c *Counter) Stop() {
//			c.act.Stop()
//		}
//
// The options for the constructor allow to pass a context for stopping,
// the timeout of actions, how many actions are queued, and a function
// for notifications in case of panics.
//
// For regular actions of the actor there are still the Interval methods.
// They allow the actions to be called at defined intervals.
//
//	func (c *Counter) startAutoIncrement() {
//	    c.act.Interval(1*time.Second, func() {
//	        c.counter++
//	    })
//	}
package actor // import "tideland.dev/go/actor"

// EOF
