# Tideland Go Actor - API Reference

This document provides a comprehensive reference for the Tideland Go Actor package API. For usage patterns and best practices, see the [package documentation](https://pkg.go.dev/tideland.dev/go/actor).

## Table of Contents

- [Creating Actors](#creating-actors)
- [Configuration](#configuration)
- [Actor Methods](#actor-methods)
  - [State Modification](#state-modification)
  - [State Reading](#state-reading)
  - [Atomic Operations](#atomic-operations)
  - [Lifecycle Management](#lifecycle-management)
  - [Repeating Actions](#repeating-actions)
  - [Monitoring](#monitoring)
- [Error Handling](#error-handling)
- [Result Type](#result-type)

---

## Creating Actors

### `actor.Go[S any](initialState S, cfg *Config) (*Actor[S], error)`

Creates and starts a new actor that owns the given initial state.

**Parameters:**
- `initialState`: The initial state of type S that the actor will own
- `cfg`: Configuration for the actor (created via `NewConfig` or `DefaultConfig`)

**Returns:**
- `*Actor[S]`: A pointer to the running actor
- `error`: Any error that occurred during actor creation

**Example:**

```go
type Account struct {
    balance      int
    holder       string
    currency     string
    transactions int
}

cfg := actor.NewConfig(context.Background())
account, err := actor.Go(Account{
    balance:      100,
    holder:       "Alice",
    currency:     "USD",
    transactions: 0,
}, cfg)
if err != nil {
    log.Fatal(err)
}
defer account.Stop()
```

---

## Configuration

### `actor.NewConfig(ctx context.Context) *Config`

Creates a new configuration with the given context and default settings.

**Parameters:**
- `ctx`: Context for lifecycle management (cancellation will stop the actor)

**Returns:**
- `*Config`: A new configuration builder

**Example:**

```go
cfg := actor.NewConfig(context.Background())
```

### `actor.DefaultConfig() *Config`

Creates a new configuration with `context.Background()` and default settings.

**Returns:**
- `*Config`: A new configuration builder with default context

**Example:**

```go
cfg := actor.DefaultConfig()
```

### Configuration Builder Methods

The following methods can be chained to customize actor behavior:

#### `.SetContext(ctx context.Context) *Config`

Sets the context for lifecycle management.

**Parameters:**
- `ctx`: The context to use

**Returns:**
- `*Config`: The configuration builder for chaining

#### `.SetQueueCapacity(capacity int) *Config`

Sets the capacity of the actor's request queue.

**Parameters:**
- `capacity`: Queue size (must be > 0, default: 256)

**Returns:**
- `*Config`: The configuration builder for chaining

**Example:**

```go
cfg := actor.NewConfig(ctx).SetQueueCapacity(512)
```

#### `.SetActionTimeout(timeout time.Duration) *Config`

Sets the maximum duration for a single action to execute.

**Parameters:**
- `timeout`: Maximum action duration (0 means no timeout, default: 0)

**Returns:**
- `*Config`: The configuration builder for chaining

**Example:**

```go
cfg := actor.NewConfig(ctx).SetActionTimeout(5 * time.Second)
```

#### `.SetShutdownTimeout(timeout time.Duration) *Config`

Sets the maximum time to wait for the actor to shut down gracefully.

**Parameters:**
- `timeout`: Maximum shutdown wait time (default: 5 seconds)

**Returns:**
- `*Config`: The configuration builder for chaining

**Example:**

```go
cfg := actor.NewConfig(ctx).SetShutdownTimeout(10 * time.Second)
```

#### `.SetFinalizer(finalizer func(error) error) *Config`

Sets a cleanup function to be called when the actor stops.

**Parameters:**
- `finalizer`: Function receiving the stop error and returning a final error

**Returns:**
- `*Config`: The configuration builder for chaining

**Example:**

```go
cfg := actor.NewConfig(ctx).SetFinalizer(func(err error) error {
    log.Printf("Actor stopped: %v", err)
    // Perform cleanup (e.g., save state to database)
    return nil
})
```

#### `.Validate() error`

Validates the configuration and returns any accumulated errors.

**Returns:**
- `error`: Validation error, or nil if configuration is valid

**Example:**

```go
cfg := actor.NewConfig(ctx).
    SetQueueCapacity(512).
    SetActionTimeout(5 * time.Second)

if err := cfg.Validate(); err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

---

## Actor Methods

### State Modification

These methods modify the actor's state. They execute sequentially in the actor's goroutine.

#### `.Do(action func(*S)) error`

Executes an action synchronously. Blocks until the action completes.

**Parameters:**
- `action`: Function that receives a pointer to the state for modification

**Returns:**
- `error`: Error if the actor is stopped or context is canceled

**Example:**

```go
err := account.Do(func(s *Account) {
    s.balance += 100
    s.transactions++
})
```

#### `.DoAsync(action func(*S)) error`

Queues an action asynchronously. Returns immediately after queueing.

**Parameters:**
- `action`: Function that receives a pointer to the state for modification

**Returns:**
- `error`: Error if the action cannot be queued (actor stopped, queue full)

**Example:**

```go
err := account.DoAsync(func(s *Account) {
    s.balance += 100
    s.transactions++
})
```

#### `.DoWithError(action func(*S) error) error`

Executes an action synchronously with error handling. Blocks until the action completes.

**Parameters:**
- `action`: Function that receives a pointer to the state and returns an error

**Returns:**
- `error`: Error from the action, or actor error

**Example:**

```go
err := account.DoWithError(func(s *Account) error {
    if s.balance < 100 {
        return fmt.Errorf("insufficient funds")
    }
    s.balance -= 100
    s.transactions++
    return nil
})
```

#### `.DoAsyncWithError(action func(*S) error) error`

Queues an action asynchronously with error handling. If the action returns an error, the actor stops.

**Parameters:**
- `action`: Function that receives a pointer to the state and returns an error

**Returns:**
- `error`: Error if the action cannot be queued

**Example:**

```go
err := account.DoAsyncWithError(func(s *Account) error {
    if s.balance < 0 {
        return fmt.Errorf("invalid state: negative balance")
    }
    return nil
})
```

#### `.DoAsyncAwait(action func(*S)) func() error`

Queues an action asynchronously and returns an awaiter function. The awaiter blocks until the action completes.

**Parameters:**
- `action`: Function that receives a pointer to the state for modification

**Returns:**
- `func() error`: Awaiter function that blocks until completion

**Example:**

```go
await := account.DoAsyncAwait(func(s *Account) {
    s.balance += 100
    s.transactions++
})

// Do other work...

// Now wait for completion
err := await()
```

#### `.DoAsyncAwaitWithError(action func(*S) error) func() error`

Queues an action asynchronously with error handling and returns an awaiter function.

**Parameters:**
- `action`: Function that receives a pointer to the state and returns an error

**Returns:**
- `func() error`: Awaiter function that blocks until completion and returns action error

**Example:**

```go
await := account.DoAsyncAwaitWithError(func(s *Account) error {
    if s.balance < 100 {
        return fmt.Errorf("insufficient funds")
    }
    s.balance -= 100
    s.transactions++
    return nil
})

// Do other work...

// Now wait and check result
if err := await(); err != nil {
    log.Printf("Withdrawal failed: %v", err)
}
```

### State Reading

#### `.Query(query func(*S) any) (any, error)`

Executes a read-only query on the state. Blocks until the query completes.

**Parameters:**
- `query`: Function that receives a pointer to the state and returns a value

**Returns:**
- `any`: The value returned by the query function
- `error`: Error if the actor is stopped or context is canceled

**Example:**

```go
balance, err := account.Query(func(s *Account) any {
    return s.balance
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Balance: %d\n", balance.(int))
```

**Best Practice:** Wrap queries in type-safe methods:

```go
func (a *Account) Balance() (int, error) {
    result, err := a.actor.Query(func(s *accountState) any {
        return s.balance
    })
    if err != nil {
        return 0, err
    }
    return result.(int), nil
}
```

### Atomic Operations

#### `.Update(update func(*S) (any, error)) (any, error)`

Executes an atomic read-modify-write operation. Blocks until the update completes.

**Parameters:**
- `update`: Function that receives a pointer to the state, modifies it, and returns a result

**Returns:**
- `any`: The value returned by the update function
- `error`: Error from the update function, or actor error

**Example:**

```go
oldBalance, err := account.Update(func(s *Account) (any, error) {
    old := s.balance
    s.balance = 1000
    return old, nil
})
```

**Example with validation:**

```go
withdrawn, err := account.Update(func(s *Account) (any, error) {
    if s.balance >= 100 {
        s.balance -= 100
        return true, nil
    }
    return false, fmt.Errorf("insufficient funds")
})
```

### Lifecycle Management

#### `.Stop()`

Gracefully stops the actor. Processes all queued actions before stopping.

**Example:**

```go
account.Stop()
```

#### `.Done() <-chan struct{}`

Returns a channel that is closed when the actor stops.

**Returns:**
- `<-chan struct{}`: Channel closed on actor shutdown

**Example:**

```go
<-account.Done()
fmt.Println("Actor has stopped")
```

#### `.IsRunning() bool`

Checks if the actor is currently running.

**Returns:**
- `bool`: true if running, false otherwise

**Example:**

```go
if account.IsRunning() {
    fmt.Println("Actor is running")
}
```

#### `.IsDone() bool`

Checks if the actor has stopped.

**Returns:**
- `bool`: true if stopped, false otherwise

**Example:**

```go
if account.IsDone() {
    fmt.Println("Actor has stopped")
}
```

#### `.Err() error`

Returns the error that caused the actor to stop, or nil.

**Returns:**
- `error`: Stop error, or nil

**Example:**

```go
if err := account.Err(); err != nil {
    log.Printf("Actor stopped with error: %v", err)
}
```

### Repeating Actions

#### `.Repeat(interval time.Duration, action func(*S)) func()`

Schedules an action to execute repeatedly at the specified interval.

**Parameters:**
- `interval`: Time between executions
- `action`: Function to execute periodically

**Returns:**
- `func()`: Function to call to stop the repeating action

**Example:**

```go
type Stats struct {
    checks int
}

stats, _ := actor.Go(Stats{}, cfg)

stop := stats.Repeat(1*time.Second, func(s *Stats) {
    s.checks++
    log.Printf("Health check %d", s.checks)
})

// Later, stop the repeating action
stop()
```

#### `.RepeatWithContext(ctx context.Context, interval time.Duration, action func(*S)) func()`

Schedules an action to execute repeatedly until the context is canceled.

**Parameters:**
- `ctx`: Context for cancellation
- `interval`: Time between executions
- `action`: Function to execute periodically

**Returns:**
- `func()`: Function to call to stop the repeating action

**Example:**

```go
ctx, cancel := context.WithCancel(context.Background())

stop := stats.RepeatWithContext(ctx, 1*time.Second, func(s *Stats) {
    s.checks++
})

// Cancel via context
cancel()

// Or via returned stop function
stop()
```

### Monitoring

#### `.QueueStatus() QueueStatus`

Returns information about the actor's request queue.

**Returns:**
- `QueueStatus`: Struct containing queue depth and capacity

**Example:**

```go
status := account.QueueStatus()
fmt.Printf("Queue: %d/%d\n", status.Depth, status.Capacity)
```

**QueueStatus fields:**
- `Depth int`: Current number of queued actions
- `Capacity int`: Maximum queue capacity

---

## Error Handling

### ActorError Type

The `ActorError` type provides detailed error information.

**Fields:**
- `Code ErrorCode`: The error code
- `Message string`: Human-readable error message
- `Err error`: Wrapped underlying error (may be nil)

**Methods:**
- `.Error() string`: Returns the error message
- `.Unwrap() error`: Returns the underlying error

**Example:**

```go
err := account.Do(func(s *Account) {
    s.balance += 100
})

if err != nil {
    var actorErr *actor.ActorError
    if errors.As(err, &actorErr) {
        switch actorErr.Code {
        case actor.ErrShutdown:
            log.Println("Actor is shutting down")
        case actor.ErrTimeout:
            log.Println("Action timed out")
        }
    }
}
```

### Error Codes

Constants representing different error conditions:

#### `ErrNone`

No error occurred.

#### `ErrShutdown`

The actor is shutting down and cannot accept new actions.

#### `ErrTimeout`

An action exceeded its timeout duration.

#### `ErrCanceled`

The operation was canceled (context cancellation).

#### `ErrPanic`

A panic occurred during action execution. The actor stops when this occurs.

#### `ErrInvalid`

Invalid parameters were provided.

**Example:**

```go
switch actorErr.Code {
case actor.ErrShutdown:
    // Handle shutdown
case actor.ErrTimeout:
    // Handle timeout
case actor.ErrCanceled:
    // Handle cancellation
case actor.ErrPanic:
    // Handle panic (actor has stopped)
case actor.ErrInvalid:
    // Handle invalid parameters
}
```

---

## Result Type

### `Result[T any]`

Generic container for operation results with error handling.

**Fields:**
- `Value T`: The result value
- `Err error`: Error if the operation failed

**Methods:**

#### `.IsOK() bool`

Returns true if there is no error.

#### `.IsErr() bool`

Returns true if there is an error.

#### `.Unwrap() (T, error)`

Returns the value and error.

**Example:**

```go
type Result[T any] struct {
    Value T
    Err   error
}

// Usage in custom types
func (a *Account) WithdrawResult(amount int) actor.Result[bool] {
    result, err := a.actor.Update(func(s *accountState) (any, error) {
        if s.balance < amount {
            return false, fmt.Errorf("insufficient funds")
        }
        s.balance -= amount
        return true, nil
    })
    
    if err != nil {
        return actor.Result[bool]{Err: err}
    }
    
    return actor.Result[bool]{Value: result.(bool)}
}

// Caller
result := account.WithdrawResult(100)
if result.IsOK() {
    fmt.Printf("Withdrawal successful: %v\n", result.Value)
} else {
    log.Printf("Error: %v\n", result.Err)
}
```

---

## Thread Safety

All actor methods are thread-safe and can be called concurrently from multiple goroutines. The actor ensures that:

1. All state access is serialized
2. Actions execute in the order they are queued
3. No data races can occur
4. State cannot be accessed directly (compiler-enforced)

**Example - Concurrent Safety:**

```go
account, _ := actor.Go(Account{balance: 0}, cfg)

// Launch 100 goroutines
for i := 0; i < 100; i++ {
    go func() {
        for j := 0; j < 10; j++ {
            account.DoAsync(func(s *Account) {
                s.balance += 10
                s.transactions++
            })
        }
    }()
}

time.Sleep(100 * time.Millisecond)

balance, _ := account.Query(func(s *Account) any {
    return s.balance
})
// Always exactly 10000 - no race conditions possible!
fmt.Printf("Balance: %d\n", balance.(int))
```
