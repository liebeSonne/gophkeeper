package service

import "errors"

var (
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrDataNotFound            = errors.New("data not found")
	ErrDataAccessDenied        = errors.New("data access denied")
	ErrFileNotFound            = errors.New("file not found")
	ErrFileAccessDenied        = errors.New("file access denied")
	ErrInvalidChunkIndex       = errors.New("invalid chunk index")
	ErrFileUploadNotInProgress = errors.New("file upload not in progress")
	ErrInvalidChinksCount      = errors.New("invalid chunks count (must be at least 1)")
	ErrFileNotCompleted        = errors.New("file not completed")
	ErrFileReferenceInvalid    = errors.New("one or more file references are invalid")
)
