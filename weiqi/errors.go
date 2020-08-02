package weiqi

import (
	"errors"
	"fmt"
)

// ErrWrongPlayer means that it is the other player's turn
var ErrWrongPlayer error = errors.New("wrong player")

// ErrOutsideBoard means that the vertex exceeds the size of the board
var ErrOutsideBoard error = errors.New("outside board")

// ErrVertexNotEmpty means that there is already a stone at the vertex
var ErrVertexNotEmpty error = errors.New("vertex not empty")

// ErrSuicide means that the move is suicidal
var ErrSuicide error = errors.New("suicide")

// ErrSituationalSuperko means that the same position has been created by the same player before
var ErrSituationalSuperko error = errors.New("violates situational superko")

// ErrPositionalSuperko means that the same position has been created before
var ErrPositionalSuperko error = errors.New("violates positional superko")

// GameError wraps an error with additional information about the attempted move
type GameError struct {
	err       error
	attempted Move
}

func (e GameError) Error() string {
	return fmt.Sprintf("move %q invalid: %s", e.attempted, e.err)
}

func (e GameError) Unwrap() error {
	return e.err
}
