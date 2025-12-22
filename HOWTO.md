# How to Use Tideland Go Actor

This guide describes the **intended usage pattern** for building concurrent-safe types using the Actor package.

## The Pattern

The recommended approach is to create a **wrapper struct** that contains an Actor and exposes convenient public methods to users. These public methods internally leverage the Actor's powerful concurrency primitives.

### Why This Pattern?

- **Clean API**: Users interact with intuitive, domain-specific methods
- **Encapsulation**: The Actor implementation detail is hidden
- **Type Safety**: Return concrete types instead of `any`
- **Thread Safety**: All state access is automatically serialized
- **No Boilerplate**: Users don't need to write closures for every operation

## Basic Example: Thread-Safe Counter

Here's how to build a concurrent-safe counter:

```go
package main

import (
    "context"
    "tideland.dev/go/actor"
)

// counterState is the internal state owned by the actor
type counterState struct {
    value int
}

// Counter provides a thread-safe counter
type Counter struct {
    actor *actor.Actor[counterState]
}

// NewCounter creates a new thread-safe counter
func NewCounter(ctx context.Context) (*Counter, error) {
    cfg := actor.NewConfig(ctx)
    act, err := actor.Go(counterState{value: 0}, cfg)
    if err != nil {
        return nil, err
    }

    return &Counter{actor: act}, nil
}

// Increment adds 1 to the counter
func (c *Counter) Increment() error {
    return c.actor.Do(func(s *counterState) {
        s.value++
    })
}

// Add adds n to the counter
func (c *Counter) Add(n int) error {
    return c.actor.Do(func(s *counterState) {
        s.value += n
    })
}

// Value returns the current counter value
func (c *Counter) Value() (int, error) {
    result, err := c.actor.Query(func(s *counterState) any {
        return s.value
    })
    if err != nil {
        return 0, err
    }
    return result.(int), nil
}

// Close stops the actor
func (c *Counter) Close() {
    c.actor.Stop()
}

// Usage
func main() {
    counter, _ := NewCounter(context.Background())
    defer counter.Close()

    counter.Increment()
    counter.Add(5)

    value, _ := counter.Value()
    println("Counter:", value) // Output: Counter: 6
}
```

## Advanced Example: Bank Account

A more complex example with validation and error handling:

```go
package banking

import (
    "context"
    "fmt"
    "tideland.dev/go/actor"
)

// accountState is the internal state
type accountState struct {
    balance int
    holder  string
}

// Account provides thread-safe banking operations
type Account struct {
    actor *actor.Actor[accountState]
}

// NewAccount creates a new bank account
func NewAccount(ctx context.Context, holder string, initialBalance int) (*Account, error) {
    if initialBalance < 0 {
        return nil, fmt.Errorf("initial balance cannot be negative")
    }

    cfg := actor.NewConfig(ctx)
    act, err := actor.Go(accountState{
        balance: initialBalance,
        holder:  holder,
    }, cfg)
    if err != nil {
        return nil, err
    }

    return &Account{actor: act}, nil
}

// Deposit adds money to the account
func (a *Account) Deposit(amount int) error {
    if amount <= 0 {
        return fmt.Errorf("deposit amount must be positive")
    }

    return a.actor.Do(func(s *accountState) {
        s.balance += amount
    })
}

// Withdraw removes money from the account with validation
func (a *Account) Withdraw(amount int) error {
    if amount <= 0 {
        return fmt.Errorf("withdrawal amount must be positive")
    }

    return a.actor.DoWithError(func(s *accountState) error {
        if s.balance < amount {
            return fmt.Errorf("insufficient funds: have %d, need %d",
                s.balance, amount)
        }
        s.balance -= amount
        return nil
    })
}

// Balance returns the current balance
func (a *Account) Balance() (int, error) {
    result, err := a.actor.Query(func(s *accountState) any {
        return s.balance
    })
    if err != nil {
        return 0, err
    }
    return result.(int), nil
}

// Holder returns the account holder name
func (a *Account) Holder() (string, error) {
    result, err := a.actor.Query(func(s *accountState) any {
        return s.holder
    })
    if err != nil {
        return "", err
    }
    return result.(string), nil
}

// Transfer moves money to another account atomically
func (a *Account) Transfer(to *Account, amount int) error {
    if amount <= 0 {
        return fmt.Errorf("transfer amount must be positive")
    }

    // Withdraw from this account
    if err := a.Withdraw(amount); err != nil {
        return fmt.Errorf("transfer failed: %w", err)
    }

    // Deposit to target account
    if err := to.Deposit(amount); err != nil {
        // Rollback: add money back
        _ = a.Deposit(amount)
        return fmt.Errorf("transfer failed: %w", err)
    }

    return nil
}

// Close stops the actor
func (a *Account) Close() {
    a.actor.Stop()
}
```

Usage:

```go
alice, _ := NewAccount(ctx, "Alice", 1000)
bob, _ := NewAccount(ctx, "Bob", 500)
defer alice.Close()
defer bob.Close()

// Transfer money
if err := alice.Transfer(bob, 200); err != nil {
    log.Fatal(err)
}

aliceBalance, _ := alice.Balance()
bobBalance, _ := bob.Balance()
fmt.Printf("Alice: %d, Bob: %d\n", aliceBalance, bobBalance)
// Output: Alice: 800, Bob: 700
```

## Using Async Operations

For operations that don't need immediate results:

```go
// IncrementAsync queues an increment without waiting
func (c *Counter) IncrementAsync() error {
    return c.actor.DoAsync(func(s *counterState) {
        s.value++
    })
}

// IncrementAndWait queues an increment and returns an awaiter
func (c *Counter) IncrementAndWait() func() error {
    return c.actor.DoAsyncAwait(func(s *counterState) {
        s.value++
    })
}

// Usage
await := counter.IncrementAndWait()
// ... do other work ...
err := await() // Now wait for completion
```

## Background Operations with Repeat

For periodic operations like health checks or cleanup:

```go
type Cache struct {
    actor     *actor.Actor[cacheState]
    stopClean func()
}

type cacheState struct {
    items map[string]cacheItem
}

type cacheItem struct {
    value     any
    expiresAt time.Time
}

func NewCache(ctx context.Context) (*Cache, error) {
    cfg := actor.NewConfig(ctx)
    act, err := actor.Go(cacheState{
        items: make(map[string]cacheItem),
    }, cfg)
    if err != nil {
        return nil, err
    }

    cache := &Cache{actor: act}

    // Clean expired items every minute
    cache.stopClean = act.Repeat(1*time.Minute, func(s *cacheState) {
        now := time.Now()
        for key, item := range s.items {
            if now.After(item.expiresAt) {
                delete(s.items, key)
            }
        }
    })

    return cache, nil
}

func (c *Cache) Close() {
    c.stopClean() // Stop background cleanup
    c.actor.Stop()
}
```

## Configuration Options

Customize actor behavior during creation:

```go
func NewAccount(ctx context.Context, holder string) (*Account, error) {
    cfg := actor.NewConfig(ctx).
        SetQueueCapacity(1024).                   // Larger queue for high throughput
        SetActionTimeout(5 * time.Second).        // Timeout for slow operations
        SetFinalizer(func(err error) error {      // Cleanup on shutdown
            log.Printf("Account %s closed: %v", holder, err)
            // Could save state to database here
            return nil
        })

    act, err := actor.Go(accountState{holder: holder}, cfg)
    if err != nil {
        return nil, err
    }

    return &Account{actor: act}, nil
}
```

## Best Practices

### 1. Keep State Private

The state struct should be unexported (lowercase) to prevent direct access:

```go
// ✅ Good - state is private
type counterState struct {
    value int
}

type Counter struct {
    actor *actor.Actor[counterState]
}

// ❌ Bad - state is public
type Counter struct {
    actor *actor.Actor[CounterState] // Now users could access state!
}
```

### 2. Return Concrete Types

Convert `any` returns to concrete types in your public methods:

```go
// ✅ Good - returns concrete type
func (c *Counter) Value() (int, error) {
    result, err := c.actor.Query(func(s *counterState) any {
        return s.value
    })
    if err != nil {
        return 0, err
    }
    return result.(int), nil
}

// ❌ Bad - leaks any type to caller
func (c *Counter) Value() (any, error) {
    return c.actor.Query(func(s *counterState) any {
        return s.value
    })
}
```

### 3. Validate Before Actor Operations

Validate inputs before submitting work to the actor:

```go
// ✅ Good - validates before queueing work
func (a *Account) Withdraw(amount int) error {
    if amount <= 0 {
        return fmt.Errorf("amount must be positive")
    }

    return a.actor.DoWithError(func(s *accountState) error {
        if s.balance < amount {
            return fmt.Errorf("insufficient funds")
        }
        s.balance -= amount
        return nil
    })
}

// ❌ Bad - validates inside actor action
func (a *Account) Withdraw(amount int) error {
    return a.actor.DoWithError(func(s *accountState) error {
        // Validation happens after queueing - wastes actor time
        if amount <= 0 {
            return fmt.Errorf("amount must be positive")
        }
        if s.balance < amount {
            return fmt.Errorf("insufficient funds")
        }
        s.balance -= amount
        return nil
    })
}
```

### 4. Provide Close/Shutdown Methods

Always provide a way to gracefully stop the actor:

```go
func (c *Counter) Close() error {
    c.actor.Stop()
    <-c.actor.Done() // Wait for shutdown
    return c.actor.Err()
}
```

### 5. Use DoWithError for Operations That Can Fail

When operations might fail, use error-returning methods:

```go
// ✅ Good - can return validation errors
func (a *Account) Withdraw(amount int) error {
    return a.actor.DoWithError(func(s *accountState) error {
        if s.balance < amount {
            return fmt.Errorf("insufficient funds")
        }
        s.balance -= amount
        return nil
    })
}

// ❌ Bad - swallows errors, might panic
func (a *Account) Withdraw(amount int) {
    a.actor.Do(func(s *accountState) {
        if s.balance < amount {
            panic("insufficient funds") // Don't do this!
        }
        s.balance -= amount
    })
}
```

## Testing Your Actor-Based Types

Since the actor encapsulates all concurrency, testing is straightforward:

```go
func TestCounter(t *testing.T) {
    counter, err := NewCounter(context.Background())
    if err != nil {
        t.Fatal(err)
    }
    defer counter.Close()

    // Test increment
    if err := counter.Increment(); err != nil {
        t.Fatal(err)
    }

    value, err := counter.Value()
    if err != nil {
        t.Fatal(err)
    }
    if value != 1 {
        t.Errorf("expected 1, got %d", value)
    }

    // Test concurrent access
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter.Increment()
        }()
    }
    wg.Wait()

    value, _ = counter.Value()
    if value != 101 { // 1 + 100
        t.Errorf("expected 101, got %d", value)
    }
}
```

## Summary

The actor pattern provides:
- **Thread safety** without manual locking
- **Clean APIs** through wrapper types
- **Type safety** with Go generics
- **Simplicity** - users don't need to understand actors to use your code

By following this pattern, you create concurrent-safe types that are easy to use and impossible to misuse.
