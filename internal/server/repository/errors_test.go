package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	testCases := []struct {
		name         string
		err          error
		wantNotFound bool
		wantConflict bool
	}{
		{
			name:         "not found error",
			err:          ErrNotFound,
			wantNotFound: true,
			wantConflict: false,
		},
		{
			name:         "wrapped not found error",
			err:          fmt.Errorf("wrapped: %w", ErrNotFound),
			wantNotFound: true,
			wantConflict: false,
		},
		{
			name:         "conflict error",
			err:          &ErrConflict{Field: "login", Value: "test@example.com", Cause: errors.New("duplicate key")},
			wantNotFound: false,
			wantConflict: true,
		},
		{
			name:         "wrapped conflict error",
			err:          fmt.Errorf("wrapped: %w", &ErrConflict{Field: "login", Value: "test@example.com", Cause: errors.New("duplicate key")}),
			wantNotFound: false,
			wantConflict: true,
		},
		{
			name:         "regular error",
			err:          errors.New("regular error"),
			wantNotFound: false,
			wantConflict: false,
		},
		{
			name:         "nil error",
			err:          nil,
			wantNotFound: false,
			wantConflict: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantNotFound, errors.Is(tc.err, ErrNotFound))
			assert.Equal(t, tc.wantConflict, IsConflict(tc.err))
		})
	}
}

func TestErrConflict_Error(t *testing.T) {
	err := &ErrConflict{Field: "login", Value: "test@example.com", Cause: errors.New("duplicate key")}
	assert.Equal(t, "conflict on login: test@example.com", err.Error())
}

func TestErrConflict_Unwrap(t *testing.T) {
	cause := errors.New("duplicate key")
	err := &ErrConflict{Field: "login", Value: "test@example.com", Cause: cause}
	assert.Equal(t, cause, err.Unwrap())
}

func TestNewErrConflictUserLogin(t *testing.T) {
	login := "testuser@example.com"
	cause := errors.New("duplicate key")
	err := NewErrConflictUserLogin(login, cause)

	assert.True(t, IsConflict(err))
	assert.Equal(t, "login", err.Field)
	assert.Equal(t, login, err.Value)
	assert.Equal(t, cause, err.Cause)
}
