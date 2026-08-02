package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/liebeSonne/gophkeeper/internal/crypto"
	apperrors "github.com/liebeSonne/gophkeeper/internal/errors"
	"github.com/liebeSonne/gophkeeper/internal/model"
	"github.com/liebeSonne/gophkeeper/internal/repository"
	"github.com/liebeSonne/gophkeeper/internal/repository/db"
)

const defaultPageSize = 20
const maxPageSize = 100

type DataService struct {
	repo      DataRepository
	encryptor crypto.Encryptor
}

func NewDataService(repo DataRepository, encryptor crypto.Encryptor) *DataService {
	return &DataService{
		repo:      repo,
		encryptor: encryptor,
	}
}

func (s *DataService) CreateData(
	ctx context.Context,
	userID uuid.UUID,
	payloadJSON []byte,
	dataType model.DataType,
	metadata string,
) (model.Data, error) {
	encrypted, err := s.encryptor.Encrypt(payloadJSON)
	if err != nil {
		return model.Data{}, fmt.Errorf("encrypt payload: %w", err)
	}

	now := time.Now()
	data := model.Data{
		ID:        s.repo.NextID(ctx),
		UserID:    userID,
		Type:      dataType,
		Payload:   encrypted,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.repo.Store(ctx, data)
	if err != nil {
		return model.Data{}, fmt.Errorf("store data: %w", err)
	}

	return data, nil
}

func (s *DataService) GetData(ctx context.Context, id, userID uuid.UUID) (model.Data, error) {
	data, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Data{}, apperrors.ErrDataNotFound
		}
		return model.Data{}, fmt.Errorf("get data: %w", err)
	}

	if data.UserID != userID {
		return model.Data{}, apperrors.ErrDataAccessDenied
	}

	decrypted, err := s.encryptor.Decrypt(data.Payload)
	if err != nil {
		return model.Data{}, fmt.Errorf("decrypt payload: %w", err)
	}

	data.Payload = decrypted
	return data, nil
}

func (s *DataService) ListData(
	ctx context.Context,
	userID uuid.UUID,
	page, pageSize int,
	dataTypes []model.DataType,
	query *string,
) ([]model.Data, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	spec := db.ListSpec{
		UserID:    userID,
		Query:     query,
		DataTypes: dataTypes,
		Limit:     &limit,
		Offset:    &offset,
	}

	total, err := s.repo.Count(ctx, spec)
	if err != nil {
		return nil, 0, fmt.Errorf("count data: %w", err)
	}

	items, err := s.repo.List(ctx, spec)
	if err != nil {
		return nil, 0, fmt.Errorf("list data: %w", err)
	}

	for i := range items {
		decrypted, err := s.encryptor.Decrypt(items[i].Payload)
		if err != nil {
			return nil, 0, fmt.Errorf("decrypt payload: %w", err)
		}
		items[i].Payload = decrypted
	}

	return items, total, nil
}

func (s *DataService) UpdateData(
	ctx context.Context,
	id uuid.UUID,
	userID uuid.UUID,
	payloadJSON []byte,
	dataType model.DataType,
	metadata string,
) (model.Data, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Data{}, apperrors.ErrDataNotFound
		}
		return model.Data{}, fmt.Errorf("get data: %w", err)
	}

	if existing.UserID != userID {
		return model.Data{}, apperrors.ErrDataAccessDenied
	}

	encrypted, err := s.encryptor.Encrypt(payloadJSON)
	if err != nil {
		return model.Data{}, fmt.Errorf("encrypt payload: %w", err)
	}

	existing.Type = dataType
	existing.Payload = encrypted
	existing.Metadata = metadata
	existing.UpdatedAt = time.Now()

	err = s.repo.Store(ctx, existing)
	if err != nil {
		return model.Data{}, fmt.Errorf("store data: %w", err)
	}

	return existing, nil
}

func (s *DataService) DeleteData(ctx context.Context, id, userID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperrors.ErrDataNotFound
		}
		return fmt.Errorf("get data: %w", err)
	}

	if existing.UserID != userID {
		return apperrors.ErrDataAccessDenied
	}

	err = s.repo.Delete(ctx, []uuid.UUID{id})
	if err != nil {
		return fmt.Errorf("delete data: %w", err)
	}

	return nil
}

func UnmarshalPayload(payload []byte, dataType model.DataType) (interface{}, error) {
	switch dataType {
	case model.DataTypeLoginPassword:
		var p model.LoginPasswordPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal login_password payload: %w", err)
		}
		return p, nil
	case model.DataTypeBankCard:
		var p model.BankCardPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal bank_card payload: %w", err)
		}
		return p, nil
	case model.DataTypeText:
		var p model.TextPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal text payload: %w", err)
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unknown data type: %d", dataType)
	}
}
