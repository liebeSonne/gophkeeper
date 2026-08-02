package repository

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

type ErrConflict struct {
	Field string
	Value string
	Cause error
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("conflict on %s: %s", e.Field, e.Value)
}

func (e *ErrConflict) Unwrap() error {
	return e.Cause
}

func NewErrConflictUserLogin(login string, cause error) *ErrConflict {
	return &ErrConflict{Field: "login", Value: login, Cause: cause}
}

func IsConflict(err error) bool {
	var conflict *ErrConflict
	return errors.As(err, &conflict)
}
