# Tideland Go Actor

[![GitHub release](https://img.shields.io/github/release/tideland/go-actor.svg)](https://github.com/tideland/go-actor)
[![GitHub license](https://img.shields.io/badge/license-New%20BSD-blue.svg)](https://raw.githubusercontent.com/tideland/go-actor/master/LICENSE)
[![Go Module](https://img.shields.io/github/go-mod/go-version/tideland/go-actor)](https://github.com/tideland/go-actor/blob/master/go.mod)
[![GoDoc](https://godoc.org/tideland.dev/go/actor?status.svg)](https://pkg.go.dev/mod/tideland.dev/go/actor?tab=packages)
![Workflow](https://github.com/tideland/go-actor/actions/workflows/build.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tideland/go-actor)](https://goreportcard.com/report/tideland.dev/go/actor)

## Description

**Tideland Go Actor** provides a robust implementation of the Actor model pattern for Go applications. The package ensures thread-safe operations by executing all actions sequentially in a dedicated background goroutine. This approach eliminates the need for explicit locking mechanisms while providing a clean and intuitive API for concurrent programming.

### Key Features

- **Sequential Execution**: All actions run in a dedicated goroutine
- **Flexible Operation Modes**: Support for both synchronous and asynchronous operations
- **Built-in Error Handling**: Automatic panic recovery with customizable recovery logic
- **Context Support**: Timeout and cancellation support via Go contexts
- **Periodic Tasks**: Easy setup of recurring actions
- **Graceful Shutdown**: Clean termination with optional finalizer functions
- **Queue Management**: Configurable action queue capacity
- **Zero Dependencies**: Pure Go implementation

## Installation

```bash
go get tideland.dev/go/actor
```

## Quick Start

Here's a simple thread-safe counter implementation:

```go
type Counter struct {
    value int
    act   *actor.Actor
}

func NewCounter() (*Counter, error) {
    // Create actor with default configuration
    cfg := actor.DefaultConfig()
    act, err := actor.Go(cfg)
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

## Configuration

The actor package uses a `Config` struct for configuration:

```go
type Config struct {
    // Controls actor lifetime
    Context context.Context

    // Capacity of action queue (must be positive)
    QueueCap int

    // Called when panic occurs during action execution
    Recoverer func(reason any) error

    // Called when actor stops
    Finalizer func(err error) error
}
```

### Default Configuration

Get default configuration with `DefaultConfig()`:

```go
cfg := actor.DefaultConfig()  // Returns Config with:
// - Context:   context.Background()
// - QueueCap:  256
// - Recoverer: Default panic -> error wrapper
// - Finalizer: Returns error unchanged
```

### Custom Configuration

Example with custom configuration:

```go
cfg := actor.Config{
    Context:  ctx,               // Custom context
    QueueCap: 1000,             // Larger queue
    Recoverer: func(r any) error {
        log.Printf("Panic: %v", r)
        return nil              // Continue execution
    },
    Finalizer: func(err error) error {
        // Cleanup resources
        if err != nil {
            log.Printf("Stopped with: %v", err)
        }
        return err
    },
}

act, err := actor.Go(cfg)
```

## Advanced Usage

### Context Support

Control operation timeouts and cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

err := c.act.DoSyncWithContext(ctx, func() {
    // Long-running operation
})
```

### Periodic Tasks

Automatically execute actions at intervals:

```go
func NewAutoCounter() (*Counter, error) {
    cfg := actor.DefaultConfig()
    act, err := actor.Go(cfg)
    if err != nil {
        return nil, err
    }
    c := &Counter{act: act}
    
    // Increment every second
    c.act.Repeat(time.Second, func() {
        c.value++
    })
    
    return c, nil
}
```

## Best Practices

1. **Keep Actions Small**: Design actions to be quick and focused
2. **Avoid Blocking**: Don't block indefinitely inside actions
3. **Error Handling**: Always check errors returned from actor methods
4. **Resource Management**: Always call `Stop()` when done with an actor
5. **Context Usage**: Use contexts for timeouts and cancellation

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the BSD License - see the [LICENSE](LICENSE) file for details.

## Contributors

- Frank Mueller (https://github.com/themue / https://github.com/tideland / https://themue.dev)

## Support

If you find this project helpful, please consider giving it a ⭐️ on GitHub!

For updates and announcements, follow us on Twitter [@tidelanddev](https://twitter.com/tidelanddev).