package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/liebeSonne/gophkeeper/internal/crypto"

	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
	"github.com/liebeSonne/gophkeeper/internal/server/repository/db"
)

const defaultPageSize = 20
const maxPageSize = 100

type DataService struct {
	repo         DataRepository
	encryptor    crypto.Encryptor
	fileProvider FileProvider
}

func NewDataService(repo DataRepository, encryptor crypto.Encryptor, fileProvider FileProvider) *DataService {
	return &DataService{
		repo:         repo,
		encryptor:    encryptor,
		fileProvider: fileProvider,
	}
}

func (s *DataService) CreateData(
	ctx context.Context,
	userID uuid.UUID,
	payloadJSON []byte,
	dataType model.DataType,
	metadata string,
) (model.Data, error) {
	if dataType == model.DataTypeFile {
		if err := s.validateFilePayload(ctx, userID, payloadJSON); err != nil {
			return model.Data{}, err
		}
	}

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
			return model.Data{}, ErrDataNotFound
		}
		return model.Data{}, fmt.Errorf("get data: %w", err)
	}

	if data.UserID != userID {
		return model.Data{}, ErrDataAccessDenied
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

	items, total, err := s.repo.ListWithCount(ctx, spec)
	if err != nil {
		return nil, 0, fmt.Errorf("list data with count: %w", err)
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
			return model.Data{}, ErrDataNotFound
		}
		return model.Data{}, fmt.Errorf("get data: %w", err)
	}

	if existing.UserID != userID {
		return model.Data{}, ErrDataAccessDenied
	}

	if dataType == model.DataTypeFile {
		err = s.validateFilePayloadUpdate(ctx, userID, existing.Payload, payloadJSON)
		if err != nil {
			return model.Data{}, err
		}
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
			return ErrDataNotFound
		}
		return fmt.Errorf("get data: %w", err)
	}

	if existing.UserID != userID {
		return ErrDataAccessDenied
	}

	err = s.repo.Delete(ctx, []uuid.UUID{id})
	if err != nil {
		return fmt.Errorf("delete data: %w", err)
	}

	return nil
}

func (s *DataService) validateFilePayload(ctx context.Context, userID uuid.UUID, payloadJSON []byte) error {
	var fp model.FilePayload
	err := json.Unmarshal(payloadJSON, &fp)
	if err != nil {
		return fmt.Errorf("unmarshal file payload: %w", err)
	}

	if len(fp.FileIDs) == 0 {
		return ErrFileReferenceInvalid
	}

	validIDs, err := s.fileProvider.GetExistingFilesByUserID(ctx, userID, fp.FileIDs)
	if err != nil {
		return fmt.Errorf("validate file references: %w", err)
	}

	if len(validIDs) != len(fp.FileIDs) {
		return ErrFileReferenceInvalid
	}

	return nil
}

func (s *DataService) validateFilePayloadUpdate(ctx context.Context, userID uuid.UUID, oldPayloadEncrypted, newPayloadJSON []byte) error {
	oldDecrypted, err := s.encryptor.Decrypt(oldPayloadEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt old payload: %w", err)
	}

	var fpOld model.FilePayload
	err = json.Unmarshal(oldDecrypted, &fpOld)
	if err != nil {
		return fmt.Errorf("unmarshal old file payload: %w", err)
	}

	var fpNew model.FilePayload
	err = json.Unmarshal(newPayloadJSON, &fpNew)
	if err != nil {
		return fmt.Errorf("unmarshal file payload: %w", err)
	}

	if fileIDsEqual(fpOld.FileIDs, fpNew.FileIDs) {
		return nil
	}

	err = s.validateFilePayload(ctx, userID, newPayloadJSON)
	if err != nil {
		return fmt.Errorf("validate new file payload: %w", err)
	}

	return nil
}

func fileIDsEqual(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[uuid.UUID]bool, len(a))
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			return false
		}
	}
	return true
}

func unmarshalPayload(payload []byte, dataType model.DataType) (interface{}, error) {
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
	case model.DataTypeFile:
		var p model.FilePayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal file payload: %w", err)
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unknown data type: %s", dataType)
	}
}
