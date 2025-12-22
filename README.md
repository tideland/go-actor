# Tideland Go Actor

[![GitHub release](https://img.shields.io/github/release/tideland/go-actor.svg)](https://github.com/tideland/go-actor)
[![GitHub license](https://img.shields.io/badge/license-New%20BSD-blue.svg)](https://raw.githubusercontent.com/tideland/go-actor/master/LICENSE)
[![Go Module](https://img.shields.io/github/go-mod/go-version/tideland/go-actor)](https://github.com/tideland/go-actor/blob/master/go.mod)
[![GoDoc](https://godoc.org/tideland.dev/go/actor?status.svg)](https://pkg.go.dev/mod/tideland.dev/go/actor?tab=packages)
![Workflow](https://github.com/tideland/go-actor/actions/workflows/build.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tideland/go-actor)](https://goreportcard.com/report/tideland.dev/go/actor)

## 📖 [Usage Guide: How to Use Tideland Go Actor](HOWTO.md)

**For the recommended usage pattern** (wrapping actors in your own types with convenient methods), **see the [HOWTO.md](HOWTO.md) guide**.

---

## API Description

**Tideland Go Actor** provides a robust implementation of the Actor Model in Go using generics to truly encapsulate state. Following the Erlang/OTP process model, actors OWN their state and only allow access through serialized message passing (closures). This makes race conditions **impossible by design** since there's no way to bypass the actor's serialization.

### Why This Design?

Traditional actor patterns in Go often embed an actor within a struct to protect fields. However, this relies on developer discipline - it's easy to accidentally write direct getters/setters that bypass the actor, creating race conditions.

**This implementation solves that problem**: The actor owns the state using Go generics. The compiler prevents direct access, making concurrent safety foolproof.

### Features

- **True Encapsulation**: State is owned by the actor, not accessible from outside
- **Type Safety**: Uses Go generics for type-safe state access without reflection
- **Impossible Race Conditions**: Compiler-enforced serialization of all state access
- **Fluent Configuration**: Worker-style builder pattern with error accumulation
- **Synchronous & Asynchronous Actions**: `Do()` blocks, `DoAsync()` queues immediately
- **Query & Update**: Convenient methods for read-only and read-modify-write operations
- **Context Integration**: First-class support for context-based cancellation
- **Repeating Actions**: Built-in support for periodic execution
- **No Panic Recovery**: Panics crash the actor (as they should in Go) rather than continuing with corrupt state
- **Zero Dependencies**: Pure Go implementation

### Installation

```bash
go get tideland.dev/go/actor
```

### Quick Start

```go
package main

import (
	"context"
	"fmt"

	"tideland.dev/go/actor"
)

func main() {
	// Define your state type
	type Counter struct {
		value int
	}

	// Create an actor that owns the state
	cfg := actor.NewConfig(context.Background())
	counter, err := actor.Go(Counter{value: 0}, cfg)
	if err != nil {
		panic(err)
	}
	defer counter.Stop()

	// Modify state (synchronous)
	counter.Do(func(s *Counter) {
		s.value++
	})

	// Read state
	value, _ := counter.Query(func(s *Counter) int {
		return s.value
	})

	fmt.Printf("Counter: %d\n", value)
}
```

### Examples

#### Basic Counter

The actor owns the state - there's NO way to access it except through the actor:

```go
type Counter struct {
	value int
}

cfg := actor.NewConfig(context.Background())
counter, _ := actor.Go(Counter{value: 0}, cfg)
defer counter.Stop()

// ✅ The ONLY way to modify state
counter.Do(func(s *Counter) {
	s.value++
})

// ❌ IMPOSSIBLE - compiler error!
// counter.value++

// ✅ The ONLY way to read state
value, _ := counter.Query(func(s *Counter) int {
	return s.value
})
```

#### Bank Account with Validation

```go
type Account struct {
	balance int
	name    string
}

cfg := actor.NewConfig(context.Background())
account, _ := actor.Go(Account{balance: 100, name: "Savings"}, cfg)
defer account.Stop()

// Deposit
account.Do(func(s *Account) {
	s.balance += 50
})

// Withdraw with validation using Update
withdrawn, err := account.Update(func(s *Account) (any, error) {
	if s.balance >= 30 {
		s.balance -= 30
		return true, nil
	}
	return false, fmt.Errorf("insufficient funds")
})

fmt.Printf("Withdrawn: %v, Error: %v\n", withdrawn, err)
```

#### Configuration with Fluent Builder

```go
cfg := actor.NewConfig(ctx).
	SetQueueCapacity(512).                    // Request queue size
	SetActionTimeout(5 * time.Second).        // Max action duration
	SetShutdownTimeout(10 * time.Second).     // Max shutdown wait
	SetFinalizer(func(err error) error {      // Cleanup on stop
		log.Printf("Actor stopped: %v", err)
		return nil
	})

// Errors are accumulated and can be checked
if err := cfg.Validate(); err != nil {
	log.Fatal(err)
}

actor, err := actor.Go(MyState{}, cfg)
```

#### Concurrent Safety

Guaranteed correctness even with heavy concurrency:

```go
type Counter struct {
	value int
}

counter, _ := actor.Go(Counter{value: 0}, cfg)
defer counter.Stop()

// Launch 100 goroutines, each incrementing 10 times
for i := 0; i < 100; i++ {
	go func() {
		for j := 0; j < 10; j++ {
			counter.DoAsync(func(s *Counter) {
				s.value++
			})
		}
	}()
}

time.Sleep(100 * time.Millisecond)

// Always exactly 1000 - no race conditions possible!
value, _ := counter.Query(func(s *Counter) int {
	return s.value
})
fmt.Printf("Value: %d\n", value) // Output: Value: 1000
```

#### Repeating Actions

```go
type Stats struct {
	healthChecks int
}

stats, _ := actor.Go(Stats{}, cfg)
defer stats.Stop()

// Run health check every second
stop := stats.Repeat(1*time.Second, func(s *Stats) {
	s.healthChecks++
	log.Printf("Health check %d", s.healthChecks)
})

// Stop the repeat when done
defer stop()
```

#### Synchronous vs Asynchronous

```go
// Synchronous - blocks until complete
err := actor.Do(func(s *State) {
	s.value = 42
})

// Asynchronous - queues and returns immediately
err = actor.DoAsync(func(s *State) {
	s.value = 42
})

// With error handling
err = actor.DoWithError(func(s *State) error {
	if s.value < 0 {
		return fmt.Errorf("invalid value")
	}
	return nil
})
```

### API Reference

#### Creating Actors

- `actor.Go[S](initialState S, cfg *Config) (*Actor[S], error)` - Create and start an actor

#### Configuration

- `actor.NewConfig(ctx context.Context) *Config` - Create new configuration
- `actor.DefaultConfig() *Config` - Create configuration with defaults
- `.SetQueueCapacity(int)` - Set request queue size (default: 256)
- `.SetActionTimeout(duration)` - Set action timeout (default: none)
- `.SetShutdownTimeout(duration)` - Set shutdown timeout (default: 5s)
- `.SetFinalizer(func(error) error)` - Set cleanup function
- `.Validate() error` - Check for configuration errors

#### Actor Methods

**State Modification:**
- `.Do(func(*S))` - Execute action synchronously
- `.DoAsync(func(*S))` - Queue action asynchronously
- `.DoWithError(func(*S) error)` - Execute with error handling
- `.DoAsyncWithError(func(*S) error)` - Queue with error handling

**State Reading:**
- `.Query(func(*S) any) (any, error)` - Read state synchronously

**Atomic Operations:**
- `.Update(func(*S) (any, error)) (any, error)` - Read-modify-write atomically

**Lifecycle:**
- `.Stop()` - Gracefully stop the actor
- `.Done() <-chan struct{}` - Channel closed when actor stops
- `.IsRunning() bool` - Check if actor is running
- `.IsDone() bool` - Check if actor has stopped
- `.Err() error` - Get error that stopped the actor

**Repeating:**
- `.Repeat(interval, func(*S)) func()` - Schedule periodic execution
- `.RepeatWithContext(ctx, interval, func(*S)) func()` - With cancellation

**Monitoring:**
- `.QueueStatus() QueueStatus` - Get queue depth and capacity

### Contributing

Contributions are welcome! Please open an issue or submit a pull request.

### License

Tideland Go Actor is licensed under the [New BSD License](LICENSE).
