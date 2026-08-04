package handler

import (
	"context"
	"io"

	"github.com/google/uuid"

	"github.com/liebeSonne/gophkeeper/internal/model"
)

type AuthService interface {
	Register(ctx context.Context, login, password string) (model.Token, error)
	Login(ctx context.Context, login, password string) (model.Token, error)
	RefreshToken(ctx context.Context, refreshToken string) (model.Token, error)
	RevokeToken(ctx context.Context, refreshToken string) error
	GetUserIDFromToken(tokenString string) (uuid.UUID, error)
}

type DataService interface {
	CreateData(ctx context.Context, userID uuid.UUID, payloadJSON []byte, dataType model.DataType, metadata string) (model.Data, error)
	GetData(ctx context.Context, id, userID uuid.UUID) (model.Data, error)
	ListData(ctx context.Context, userID uuid.UUID, page, pageSize int, dataTypes []model.DataType, query *string) ([]model.Data, int, error)
	UpdateData(ctx context.Context, id, userID uuid.UUID, payloadJSON []byte, dataType model.DataType, metadata string) (model.Data, error)
	DeleteData(ctx context.Context, id, userID uuid.UUID) error
}

type FileService interface {
	InitUpload(ctx context.Context, userID uuid.UUID, name, mimeType string, size int64, chunksCount int) (model.File, error)
	UploadChunk(ctx context.Context, fileID, userID uuid.UUID, chunkIndex int, data []byte) error
	CompleteUpload(ctx context.Context, fileID, userID uuid.UUID) (model.File, error)
	DownloadFile(ctx context.Context, fileID, userID uuid.UUID) (reader io.ReadCloser, mimeType string, size int64, err error)
	DeleteFile(ctx context.Context, fileID, userID uuid.UUID) error
	ListFiles(ctx context.Context, userID uuid.UUID, page, pageSize int, query *string) ([]model.File, int, error)
}
