// Tideland Go Actor
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

/*
Package actor provides a robust implementation of Tony Hoare's Actor Model in Go using generics
to encapsulate state. Actors maintain encapsulated state that can only be accessed through
serialized messages. In this implementation, messages are function types (closures) that receive
a pointer to the state.

This approach eliminates race conditions by design - since the state is owned by the actor and
accessed only through serialized message passing, concurrent access becomes impossible.

# The Recommended Pattern

The recommended approach is to create a wrapper struct that contains an Actor and exposes
convenient public methods. This provides a clean API while hiding the actor implementation details.

Why This Pattern?

  - Clean API: Users interact with intuitive, domain-specific methods
  - Encapsulation: The actor implementation detail is hidden
  - Type Safety: Return concrete types instead of any
  - Thread Safety: All state access is automatically serialized

# Example: Bank Account

Here's how to build a concurrent-safe bank account:

	package banking

	import (
		"context"
		"fmt"
		"tideland.dev/go/actor"
	)

	// accountState is the internal state owned by the actor
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
				return fmt.Errorf("insufficient funds")
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

	// Close stops the actor
	func (a *Account) Close() {
		a.actor.Stop()
	}

Usage:

	alice, _ := NewAccount(ctx, "Alice", 1000)
	defer alice.Close()

	alice.Deposit(200)
	alice.Withdraw(100)

	balance, _ := alice.Balance()
	fmt.Printf("Balance: %d\n", balance)

# Asynchronous Operations

For operations that don't need immediate results:

	// DepositAsync queues a deposit without waiting
	func (a *Account) DepositAsync(amount int) error {
		if amount <= 0 {
			return fmt.Errorf("amount must be positive")
		}
		return a.actor.DoAsync(func(s *accountState) {
			s.balance += amount
		})
	}

	// WithdrawAndWait queues a withdrawal and returns an awaiter
	func (a *Account) WithdrawAndWait(amount int) func() error {
		if amount <= 0 {
			return func() error { return fmt.Errorf("amount must be positive") }
		}
		return a.actor.DoAsyncAwaitWithError(func(s *accountState) error {
			if s.balance < amount {
				return fmt.Errorf("insufficient funds")
			}
			s.balance -= amount
			return nil
		})
	}

	// Usage
	await := account.WithdrawAndWait(100)
	// ... do other work ...
	err := await() // Now wait for completion

# Background Operations

For periodic operations like interest calculation:

	import "time"

	type SavingsAccount struct {
		actor         *actor.Actor[savingsState]
		stopInterest  func()
	}

	type savingsState struct {
		balance      int
		interestRate float64
	}

	func NewSavingsAccount(ctx context.Context, initialBalance int, rate float64) (*SavingsAccount, error) {
		cfg := actor.NewConfig(ctx)
		act, err := actor.Go(savingsState{
			balance:      initialBalance,
			interestRate: rate,
		}, cfg)
		if err != nil {
			return nil, err
		}

		account := &SavingsAccount{actor: act}

		// Calculate interest monthly
		account.stopInterest = act.Repeat(30*24*time.Hour, func(s *savingsState) {
			interest := int(float64(s.balance) * s.interestRate)
			s.balance += interest
		})

		return account, nil
	}

	func (s *SavingsAccount) Close() {
		s.stopInterest() // Stop background interest calculation
		s.actor.Stop()
	}

# Best Practices

Keep State Private

The state struct should be unexported (lowercase):

	// ✅ Good
	type accountState struct {
		balance int
	}

	type Account struct {
		actor *actor.Actor[accountState]
	}

Return Concrete Types

Convert any returns to concrete types in public methods:

	// ✅ Good
	func (a *Account) Balance() (int, error) {
		result, err := a.actor.Query(func(s *accountState) any {
			return s.balance
		})
		if err != nil {
			return 0, err
		}
		return result.(int), nil
	}

Validate Before Actor Operations

Validate inputs before submitting work to the actor:

	// ✅ Good
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

Provide Close/Shutdown Methods

Always provide a way to gracefully stop the actor:

	func (a *Account) Close() error {
		a.actor.Stop()
		<-a.actor.Done()
		return a.actor.Err()
	}

Use DoWithError for Operations That Can Fail

When operations might fail, use error-returning methods:

	// ✅ Good
	func (a *Account) Withdraw(amount int) error {
		return a.actor.DoWithError(func(s *accountState) error {
			if s.balance < amount {
				return fmt.Errorf("insufficient funds")
			}
			s.balance -= amount
			return nil
		})
	}

# Configuration

Customize actor behavior with the fluent configuration builder:

	cfg := actor.NewConfig(ctx).
		SetQueueCapacity(1024).
		SetActionTimeout(5 * time.Second).
		SetFinalizer(func(err error) error {
			log.Printf("Account closed: %v", err)
			// Could save final state to database here
			return nil
		})

	act, err := actor.Go(accountState{}, cfg)

# Testing

Testing is straightforward since the actor encapsulates all concurrency:

	func TestAccount(t *testing.T) {
		account, err := NewAccount(context.Background(), "Alice", 1000)
		if err != nil {
			t.Fatal(err)
		}
		defer account.Close()

		// Test deposit
		if err := account.Deposit(500); err != nil {
			t.Fatal(err)
		}

		balance, _ := account.Balance()
		if balance != 1500 {
			t.Errorf("expected 1500, got %d", balance)
		}

		// Test concurrent access
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				account.Deposit(10)
			}()
		}
		wg.Wait()

		balance, _ = account.Balance()
		if balance != 2500 { // 1500 + (100 * 10)
			t.Errorf("expected 2500, got %d", balance)
		}
	}

# API Documentation

For a complete API reference, see the API.md file in the repository or visit
https://pkg.go.dev/tideland.dev/go/actor.
*/

package actor
