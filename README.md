# Tideland Go Actor

[![GitHub release](https://img.shields.io/github/release/tideland/go-actor.svg)](https://github.com/tideland/go-actor)
[![GitHub license](https://img.shields.io/badge/license-New%20BSD-blue.svg)](https://raw.githubusercontent.com/tideland/go-actor/master/LICENSE)
[![Go Module](https://img.shields.io/github/go-mod/go-version/tideland/go-actor)](https://github.com/tideland/go-actor/blob/master/go.mod)
[![GoDoc](https://godoc.org/tideland.dev/go/actor?status.svg)](https://pkg.go.dev/mod/tideland.dev/go/actor?tab=packages)
![Workflow](https://github.com/tideland/go-actor/actions/workflows/build.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tideland/go-actor)](https://goreportcard.com/report/tideland.dev/go/actor)

## Description

**Tideland Go Actor** provides running backend goroutines for the sequential execution
of anonymous functions following the actor model. The Actors can work asynchronously as
well as synchronously. Additionally the Actor provides methods for the periodical 
execution of code under control of an Actor. So background operation can be automated.

All together simplifies the implementation of concurrent code.

I hope you like it. ;)

## Example

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
	c.startAutoIncrement()
	return c, nil
}

func (c *Counter) Incr() error {
	// Increment will be done in the background.
	return c.act.DoAsync(func() {
		c.counter++
	})
}

func (c *Counter) Get() (int, error) {
	// Retrieve the current count synchronously.
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

For periodic execution of `Actions` using an `Actor` there are the `Periodical()`
methods. They allow the actions to be called at defined intervals. Like in the
example above, the `startAutoIncrement()` method is called in the constructor.
It starts a periodical `Action` which increases the counter every second.

```go
func (c *Counter) startAutoIncrement() {
	// Run counter increment asynchronously every second.
	interval := 1 * time.Second
	c.act.Periodical(interval, func() {
		c.counter++
	})
}
```

For more control `Periodical()` returns a function which can be called to
the individual periodical action. Otherwise it will be stopped togeether
with the `Actor` or based on a passed context.

## Contributors

- Frank Mueller (https://github.com/themue / https://github.com/tideland / https://tideland.dev)

