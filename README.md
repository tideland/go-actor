# Tideland Go Actor

[![GitHub release](https://img.shields.io/github/release/tideland/go-actor.svg)](https://github.com/tideland/go-actor)
[![GitHub license](https://img.shields.io/badge/license-New%20BSD-blue.svg)](https://raw.githubusercontent.com/tideland/go-actor/master/LICENSE)
[![Go Module](https://img.shields.io/github/go-mod/go-version/tideland/go-actor)](https://github.com/tideland/go-actor/blob/master/go.mod)
[![GoDoc](https://godoc.org/tideland.dev/go/actor?status.svg)](https://pkg.go.dev/mod/tideland.dev/go/actor?tab=packages)
![Workflow](https://github.com/tideland/go-actor/actions/workflows/build.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tideland/go-actor)](https://goreportcard.com/report/tideland.dev/go/actor)

**Tideland Go Actor** provides a lightweight and robust implementation of the Actor Model in Go. It simplifies concurrent programming by encapsulating state and behavior within actors. All actions on an actor's state are executed sequentially in a dedicated background goroutine, eliminating the need for manual locking and reducing the risk of race conditions.

## Features

- **Simple API**: A clean and easy-to-use API for creating and interacting with actors.
- **Synchronous & Asynchronous Actions**: Perform blocking (`DoSync`) or non-blocking (`DoAsync`) actions.
- **Context Integration**: Control the actor's lifecycle with `context.Context`.
- **Panic Recovery**: Gracefully handle panics within actor actions using a `Recoverer` function.
- **Finalization**: Perform cleanup tasks when an actor stops using a `Finalizer` function.
- **Repeating Actions**: Schedule actions to run at a regular interval with `Repeat`.
- **Zero Dependencies**: A pure Go implementation with no external dependencies.

## Installation

```bash
go get tideland.dev/go/actor
```

## Quick Start

Here's how to create and use a simple actor:

```go
package main

import (
	"fmt"
	"time"

	"tideland.dev/go/actor"
)

func main() {
	// Create an actor with the default configuration.
	act, err := actor.Go(actor.DefaultConfig())
	if err != nil {
		panic(err)
	}
	defer act.Stop()

	// Perform a synchronous action.
	act.DoSync(func() {
		fmt.Println("Hello from inside the actor!")
	})

	// Perform an asynchronous action.
	done := make(chan struct{})
	act.DoAsync(func() {
		fmt.Println("This runs in the background.")
		close(done)
	})

	// Wait for the async action to finish.
	<-done
}
```

## Examples

### Protecting Struct State

A common pattern is to embed an actor in a struct to make its methods thread-safe.
The actor serializes access to the struct's fields, so you don't need to use mutexes.

```go
type Counter struct {
	value int
	act   *actor.Actor
}

func NewCounter() (*Counter, error) {
	c := &Counter{}
	act, err := actor.Go(actor.DefaultConfig())
	if err != nil {
		return nil, err
	}
	c.act = act
	return c, nil
}

func (c *Counter) Increment() {
	// Asynchronously increment the value.
	c.act.DoAsync(func() {
		c.value++
	})
}

func (c *Counter) Value() int {
	// Synchronously read the value.
	var value int
	c.act.DoSync(func() {
		value = c.value
	})
	return value
}

func (c *Counter) Stop() {
	c.act.Stop()
}
```

### Handling Panics

You can provide a `Recoverer` function to handle panics that occur within an actor.

```go
recovered := make(chan any, 1)

cfg := actor.DefaultConfig()
cfg.Recoverer = func(reason any) error {
	recovered <- reason
	return nil // Returning nil allows the actor to continue.
}

act, _ := actor.Go(cfg)
defer act.Stop()

act.DoSync(func() {
	panic("something went wrong")
})

reason := <-recovered
fmt.Printf("Recovered from panic: %v", reason)
```

### Repeating Actions

Use `Repeat` to schedule a function to run at a regular interval.

```go
act, _ := actor.Go(actor.DefaultConfig())
defer act.Stop()

counter := 0
stop, _ := act.Repeat(10*time.Millisecond, func() {
	if counter < 5 {
		fmt.Println("Repeating...")
	}
	counter++
})

time.Sleep(60 * time.Millisecond)
stop()
```

For more examples, see the `examples_test.go` file.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

Tideland Go Actor is licensed under the [New BSD License](LICENSE).
