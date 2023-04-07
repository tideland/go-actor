# Tideland Go Actor

[![GitHub release](https://img.shields.io/github/release/tideland/go-actor.svg)](https://github.com/tideland/go-actor)
[![GitHub license](https://img.shields.io/badge/license-New%20BSD-blue.svg)](https://raw.githubusercontent.com/tideland/go-actor/master/LICENSE)
[![Go Module](https://img.shields.io/github/go-mod/go-version/tideland/go-actor)](https://github.com/tideland/go-actor/blob/master/go.mod)
[![GoDoc](https://godoc.org/tideland.dev/go/actor?status.svg)](https://pkg.go.dev/mod/tideland.dev/go/actor?tab=packages)
![Workflow](https://github.com/tideland/go-actor/actions/workflows/build.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tideland/go-actor)](https://goreportcard.com/report/tideland.dev/go/actor)

### Description

**Tideland Go Actor** provides running backend goroutines for the sequential execution
of anonymous functions following the actor model. The Actors can work asynchronously as
well as synchronously. Additionally the Actor provides methods for the repeated execution 
of Actions. So background operation can be automated.

The options for the constructor allow to pass a context for the Actor, the capacity
of the Action queue, a recoverer function in case of an Action panic and a finalizer
function when the Actor stops.

All together simplifies the implementation of concurrent code.

I hope you like it. ;)

### Example

```go
type Counter struct {
	counter int
	act     *actor.Actor
}

func NewCounter() (*Counter, error) {
	act, err := actor.Go()
	if err != nil {
		return nil, err
	}
	c := &Counter{
		counter: 0,
		act:     act,
	}
	// Increment the counter every second.
	interval := 1 * time.Second
	c.act.Repeat(interval, func() {
		c.counter++
	})
	return c, nil
}

func (c *Counter) Incr() error {
	return c.act.DoAsync(func() {
		c.counter++
	})
}

func (c *Counter) Get() (int, error) {
	var counter int
	if err := c.act.DoSync(func() {
		counter = c.counter
	}); err != nil {
		return 0, err
	}
	return counter, nil
}

func (c *Counter) Stop() {
	c.act.Stop()
}
```

### Contributors

- Frank Mueller (https://github.com/themue / https://github.com/tideland / https://tideland.dev)
