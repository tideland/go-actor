# Tideland Go Actor

[![GitHub release](https://img.shields.io/github/release/tideland/go-actor.svg)](https://github.com/tideland/go-actor)
[![GitHub license](https://img.shields.io/badge/license-New%20BSD-blue.svg)](https://raw.githubusercontent.com/tideland/go-actor/master/LICENSE)
[![Go Module](https://img.shields.io/github/go-mod/go-version/tideland/go-actor)](https://github.com/tideland/go-actor/blob/master/go.mod)
[![GoDoc](https://godoc.org/tideland.dev/go/actor?status.svg)](https://pkg.go.dev/mod/tideland.dev/go/actor?tab=packages)
![Workflow](https://github.com/tideland/go-actor/actions/workflows/build.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tideland/go-actor)](https://goreportcard.com/report/tideland.dev/go/actor)

## Description

**Tideland Go Actor** is a powerful implementation of the Actor model pattern for Go applications. It simplifies concurrent programming by providing a safe and efficient way to handle shared state without explicit locking mechanisms. Instead of dealing with mutexes and channels directly, you can focus on your business logic while the actor system ensures thread-safe execution.

### Key Features

- **Sequential Execution**: All actions run in a dedicated goroutine, eliminating race conditions
- **Flexible Operation Modes**: Support for both synchronous and asynchronous operations
- **Built-in Error Handling**: Automatic panic recovery with customizable recovery logic
- **Context Support**: Timeout and cancellation support via Go contexts
- **Periodic Tasks**: Easy setup of recurring actions with configurable intervals
- **Graceful Shutdown**: Clean termination with optional finalizer functions
- **Queue Management**: Configurable action queue capacity
- **Zero Dependencies**: Pure Go implementation with no external dependencies

## Installation

```bash
go get tideland.dev/go/actor
```

## Quick Start

Here's a simple thread-safe counter implementation using the actor package:

```go
type Counter struct {
    value int
    act   *actor.Actor
}

func NewCounter() (*Counter, error) {
    // Create a new actor with default options
    act, err := actor.Go()
    if err != nil {
        return nil, err
    }
    return &Counter{act: act}, nil
}

// Increment asynchronously increases the counter
func (c *Counter) Increment() error {
    return c.act.DoAsync(func() {
        c.value++
    })
}

// Value synchronously retrieves the current count
func (c *Counter) Value() (int, error) {
    var v int
    err := c.act.DoSync(func() {
        v = c.value
    })
    return v, err
}

// Stop terminates the actor
func (c *Counter) Stop() {
    c.act.Stop()
}
```

## Advanced Usage

### Context Support

Control operation timeouts and cancellation:

```go
func (c *Counter) IncrementWithTimeout(timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    return c.act.DoAsyncWithContext(ctx, func() {
        c.value++
    })
}
```

### Periodic Tasks

Automatically execute actions at regular intervals:

```go
func NewAutoCounter() (*Counter, error) {
    act, err := actor.Go()
    if err != nil {
        return nil, err
    }
    c := &Counter{act: act}

    // Increment every second
    interval := time.Second
    c.act.Repeat(interval, func() {
        c.value++
    })

    return c, nil
}
```

### Custom Error Recovery

Handle panics gracefully:

```go
act, err := actor.Go(
    actor.WithRecoverer(func(reason any) error {
        log.Printf("Recovered from panic: %v", reason)
        return nil // Continue execution
    }),
)
```

### Graceful Shutdown

Clean up resources on termination:

```go
act, err := actor.Go(
    actor.WithFinalizer(func(err error) error {
        // Perform cleanup
        if err != nil {
            log.Printf("Actor stopped with error: %v", err)
        }
        return err
    }),
)
```

### Queue Configuration

Control the action queue size:

```go
act, err := actor.Go(
    actor.WithQueueCap(100), // Set queue capacity to 100 actions
)
```

## Best Practices

1. **Keep Actions Small**: Design actions to be quick and focused
2. **Avoid Blocking**: Don't block indefinitely inside actions
3. **Error Handling**: Always check errors returned from actor methods
4. **Resource Management**: Always call `Stop()` when done with an actor
5. **Context Usage**: Use contexts for timeouts and cancellation in long-running operations

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the BSD License - see the [LICENSE](LICENSE) file for details.

## Contributors

- Frank Mueller (https://github.com/themue / https://github.com/tideland / https://themue.dev)

## Support

If you find this project helpful, please consider giving it a ⭐️ on GitHub!
