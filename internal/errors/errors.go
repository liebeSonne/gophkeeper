// nolint: revive
package errors

import "errors"

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrDataNotFound = errors.New("data not found")
var ErrDataAccessDenied = errors.New("data access denied")
