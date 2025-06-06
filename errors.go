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

import (
	"fmt"
)

//--------------------
// ERROR TYPES
//--------------------

// ErrorCode defines the type of error that occurred.
type ErrorCode int

const (
	// ErrNone signals no error.
	ErrNone ErrorCode = iota
	// ErrShutdown signals that the actor is shutting down.
	ErrShutdown
	// ErrTimeout signals a timeout during operation.
	ErrTimeout
	// ErrCanceled signals that the operation was canceled.
	ErrCanceled
	// ErrPanic signals that a panic occurred during action execution.
	ErrPanic
	// ErrInvalid signals invalid parameters or state.
	ErrInvalid
)

// String implements the Stringer interface.
func (ec ErrorCode) String() string {
	switch ec {
	case ErrNone:
		return "no error"
	case ErrShutdown:
		return "actor shutdown"
	case ErrTimeout:
		return "timeout"
	case ErrCanceled:
		return "canceled"
	case ErrPanic:
		return "panic"
	case ErrInvalid:
		return "invalid"
	default:
		return "unknown error"
	}
}

// ActorError contains detailed information about an actor error.
type ActorError struct {
	Op   string
	Err  error
	Code ErrorCode
}

// Error implements the error interface.
func (e *ActorError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("actor %s: %v (%v)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("actor %s: %v", e.Op, e.Code)
}

// Unwrap implements error unwrapping.
func (e *ActorError) Unwrap() error {
	return e.Err
}

// NewError creates a new actor error.
func NewError(op string, err error, code ErrorCode) *ActorError {
	return &ActorError{
		Op:   op,
		Err:  err,
		Code: code,
	}
}

// EOF
