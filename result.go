// Tideland Go Actor
//
// Copyright (C) 2019-2025 Frank Mueller / Tideland / Oldenburg / Germany
//
// All rights reserved. Use of this source code is governed
// by the new BSD license.

package actor

//--------------------
// IMPORTS
//--------------------

//--------------------
// RESULT
//--------------------

// Result encapsulates both a value and a potential error.
type Result[T any] struct {
	value T
	err   error
}

// Value returns the encapsulated value.
func (r Result[T]) Value() T {
	return r.value
}

// Err returns any error that occurred.
func (r Result[T]) Err() error {
	return r.err
}

// Ok returns true if there is no error.
func (r Result[T]) Ok() bool {
	return r.err == nil
}

// NewResult creates a new Result instance.
func NewResult[T any](value T, err error) Result[T] {
	return Result[T]{
		value: value,
		err:   err,
	}
}

// EOF
