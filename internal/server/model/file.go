package model

import (
	"time"

	"github.com/google/uuid"
)

type FileStatus string

const (
	FileStatusInProgress FileStatus = "IN_PROGRESS"
	FileStatusCompleted  FileStatus = "COMPLETED"
	FileStatusFailed     FileStatus = "FAILED"
)

type File struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Name        string     `json:"name"`
	MimeType    string     `json:"mime_type"`
	Size        int64      `json:"size"`
	ChunksCount int        `json:"chunks_count"`
	Status      FileStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type FileChunk struct {
	ID         uuid.UUID `json:"id"`
	FileID     uuid.UUID `json:"file_id"`
	ChunkIndex int       `json:"chunk_index"`
	Uploaded   bool      `json:"uploaded"`
	CreatedAt  time.Time `json:"created_at"`
}
