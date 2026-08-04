// nolint: revive
package errors

import "errors"

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrDataNotFound = errors.New("data not found")
var ErrDataAccessDenied = errors.New("data access denied")
var ErrFileNotFound = errors.New("file not found")
var ErrFileAccessDenied = errors.New("file access denied")
var ErrInvalidChunkIndex = errors.New("invalid chunk index")
var ErrFileUploadNotInProgress = errors.New("file upload not in progress")
var ErrInvalidChinksCount = errors.New("invalid chunks count (must be at least 1)")
var ErrFileNotCompleted = errors.New("file not completed")
