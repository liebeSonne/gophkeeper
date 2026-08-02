package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/liebeSonne/gophkeeper/internal/model"
)

type DataService interface {
	CreateData(ctx context.Context, userID uuid.UUID, payloadJSON []byte, dataType model.DataType, metadata string) (model.Data, error)
	GetData(ctx context.Context, id, userID uuid.UUID) (model.Data, error)
	ListData(ctx context.Context, userID uuid.UUID, page, pageSize int, dataTypes []model.DataType, query *string) ([]model.Data, int, error)
	UpdateData(ctx context.Context, id, userID uuid.UUID, payloadJSON []byte, dataType model.DataType, metadata string) (model.Data, error)
	DeleteData(ctx context.Context, id, userID uuid.UUID) error
}
