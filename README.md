# Tideland Go Actor

[![GitHub release](https://img.shields.io/github/release/tideland/go-actor.svg)](https://github.com/tideland/go-actor)
[![GitHub license](https://img.shields.io/badge/license-New%20BSD-blue.svg)](https://raw.githubusercontent.com/tideland/go-actor/master/LICENSE)
[![Go Module](https://img.shields.io/github/go-mod/go-version/tideland/go-actor)](https://github.com/tideland/go-actor/blob/master/go.mod)
[![GoDoc](https://godoc.org/tideland.dev/go/actor?status.svg)](https://pkg.go.dev/mod/tideland.dev/go/actor?tab=packages)
![Workflow](https://github.com/tideland/go-actor/actions/workflows/build.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tideland/go-actor)](https://goreportcard.com/report/tideland.dev/go/actor)

## Description

**Tideland Go Actor** provides a lightweight implementation of the Actor model pattern for Go applications. The package ensures thread-safe operations by executing all actions sequentially in a dedicated background goroutine, eliminating the need for explicit locking mechanisms.

### Key Features

- **Sequential Execution**: All actions run in a dedicated goroutine
- **Operation Modes**: Both synchronous and asynchronous execution
- **Timeout Control**: Global and per-action timeouts
- **Queue Monitoring**: Track queue status and capacity
- **Error Handling**: Built-in panic recovery
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
    cfg.ActionTimeout = 5 * time.Second  // Default timeout for all actions
    
    act, err := actor.Go(cfg)
    if err != nil {
        return nil, err
    }
    return &Counter{act: act}, nil
}

// Increment asynchronously with timeout
func (c *Counter) Increment() error {
    return c.act.DoAsyncTimeout(time.Second, func() {
        c.value++
    })
}

// Value returns current count synchronously
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

The actor package uses a `Config` struct for initialization:

```go
cfg := actor.Config{
    Context:       ctx,               // Controls actor lifetime
    QueueCap:      1000,             // Action queue capacity
    ActionTimeout: 5 * time.Second,   // Default timeout for actions
    Recoverer:     func(r any) error {
        log.Printf("Panic: %v", r)
        return nil
    },
    Finalizer:     func(err error) error {
        if err != nil {
            log.Printf("Stopped: %v", err)
        }
        return err
    },
}

act, err := actor.Go(cfg)
```

## Queue Monitoring

Monitor queue status to prevent overload:

```go
status := act.QueueStatus()
fmt.Printf("Queue: %d/%d (full: %v)\n", 
    status.Length, status.Capacity, status.IsFull)
```

## Timeout Handling

Three ways to handle timeouts:

```go
// 1. Global timeout in configuration
cfg := actor.DefaultConfig()
cfg.ActionTimeout = 5 * time.Second

// 2. Per-action timeout
err := act.DoSyncTimeout(2*time.Second, func() {
    // Operation with 2s timeout
})

// 3. Context timeout
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
err = act.DoSyncWithContext(ctx, func() {
    // Operation with context timeout
})
```

## Best Practices

1. **Keep Actions Small**: Design actions to be quick and focused
2. **Use Timeouts**: Set appropriate timeouts to prevent hanging
3. **Monitor Queue**: Check queue status to prevent overload
4. **Error Handling**: Always check returned errors
5. **Resource Management**: Call `Stop()` when done

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the BSD License - see the [LICENSE](LICENSE) file for details.

## Contributors

- Frank Mueller (https://github.com/themue / https://github.com/tideland / https://themue.dev)