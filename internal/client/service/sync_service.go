package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	apiClient "github.com/liebeSonne/gophkeeper/internal/client/adapter/gophkeeper"
	"github.com/liebeSonne/gophkeeper/internal/client/model"
	"github.com/liebeSonne/gophkeeper/internal/client/storage"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

var _ = json.Unmarshal // ensure encoding/json is used

var ErrClientNotConfigured = errors.New("client not configured")
var ErrUnexpectedCreateDataResponse = errors.New("unexpected create data response")
var ErrUnexpectedDataListResponse = errors.New("unexpected data list response")

type Service struct {
	store  *storage.Store
	client *apiClient.Client
	logger intlogger.Logger

	mu     sync.Mutex
	ticker *time.Ticker
	stop   chan struct{}
	done   chan struct{}
}

func NewService(store *storage.Store, client *apiClient.Client, logger intlogger.Logger, interval time.Duration) *Service {
	return &Service{
		store:  store,
		client: client,
		logger: logger,
		ticker: time.NewTicker(interval),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (s *Service) Start() {
	go s.run()
}

func (s *Service) Stop() {
	close(s.stop)
	<-s.done
}

func (s *Service) run() {
	defer close(s.done)
	for {
		select {
		case <-s.stop:
			s.ticker.Stop()
			return
		case <-s.ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := s.Sync(ctx)
			if err != nil {
				s.logger.Debug("sync error", "err", err)
			}
			cancel()
		}
	}
}

func (s *Service) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.store.GetTokens()
	if err != nil || tokens == nil {
		s.logger.Debug("no tokens, skipping sync")
		return nil
	}

	s.client.SetAuthToken(tokens.AccessToken)

	err = s.syncPull(ctx)
	if err != nil {
		s.logger.Warn("sync pull failed", "err", err)
	}

	err = s.syncPush(ctx)
	if err != nil {
		s.logger.Warn("sync push failed", "err", err)
	}

	return nil
}

func (s *Service) syncPush(ctx context.Context) error {
	if s.client == nil {
		return ErrClientNotConfigured
	}

	pending, err := s.store.DataGetPending()
	if err != nil {
		return fmt.Errorf("get pending entries: %w", err)
	}

	for _, entry := range pending {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.SyncStatus == model.SyncStatusDeleting {
			err = s.syncDeleteEntry(ctx, entry)
			if err != nil {
				s.logger.Warn("delete entry failed", "id", entry.ID, "err", err)
				errStatus := s.store.DataUpdateSyncStatus(entry.ID, model.SyncStatusPending, entry.RemoteID, nil, strPtr(err.Error()))
				if errStatus != nil {
					s.logger.Warn("error on update sync status to pending", "id", entry.ID, "err", errStatus)
				}
			}
			continue
		}

		err = s.syncPushEntry(ctx, entry)
		if err != nil {
			s.logger.Warn("push entry failed", "id", entry.ID, "err", err)
			errStatus := s.store.DataUpdateSyncStatus(entry.ID, model.SyncStatusPending, entry.RemoteID, nil, strPtr(err.Error()))
			if errStatus != nil {
				s.logger.Warn("error on update sync status to pending", "id", entry.ID, "err", errStatus)
			}
		}
	}

	return nil
}

func (s *Service) syncPushEntry(ctx context.Context, entry model.DataEntry) error {
	data, err := buildDataFromEntry(entry)
	if err != nil {
		return err
	}

	if entry.RemoteID == nil {
		resp, createErr := s.client.CreateData(ctx, *data)
		if createErr != nil {
			return fmt.Errorf("create data: %w", createErr)
		}

		if resp.JSON201 == nil || resp.JSON201.Data == nil {
			return ErrUnexpectedCreateDataResponse
		}

		remoteID, extractErr := extractDataID(resp.JSON201.Data)
		if extractErr != nil {
			return fmt.Errorf("extract remote ID: %w", extractErr)
		}

		etag := strPtr(time.Now().Format(time.RFC3339))
		return s.store.DataUpdateSyncStatus(entry.ID, model.SyncStatusSynced, &remoteID, etag, nil)
	}

	_, err = s.client.UpdateData(ctx, *entry.RemoteID, *data)
	if err != nil {
		return fmt.Errorf("update data: %w", err)
	}

	etag := strPtr(time.Now().Format(time.RFC3339))
	return s.store.DataUpdateSyncStatus(entry.ID, model.SyncStatusSynced, entry.RemoteID, etag, nil)
}

func (s *Service) syncDeleteEntry(ctx context.Context, entry model.DataEntry) error {
	if entry.RemoteID == nil {
		return s.store.DataDelete(entry.ID)
	}

	if err := s.client.DeleteData(ctx, *entry.RemoteID); err != nil {
		return err
	}

	return s.store.DataDelete(entry.ID)
}

func (s *Service) syncPull(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("client not configured")
	}
	count, err := s.store.DataCount()
	if err != nil {
		return fmt.Errorf("count data: %w", err)
	}

	if count > 0 {
		return nil
	}

	page := 1
	pageSize := 100

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := s.client.ListData(ctx, &page, &pageSize, nil, nil)
		if err != nil {
			s.logger.Debug("server unavailable during pull", "err", err)
			return fmt.Errorf("list data: %w", err)
		}

		if resp.JSON200 == nil {
			return ErrUnexpectedDataListResponse
		}

		for _, item := range resp.JSON200.Items {
			err = s.pullEntry(&item)
			if err != nil {
				s.logger.Warn("pull entry failed", "err", err)
			}
		}

		if page >= resp.JSON200.TotalPages {
			break
		}
		page++
	}

	return nil
}

func (s *Service) pullEntry(item *gophkeeper.DataInfo) error {
	entry, err := buildEntryFromData(item)
	if err != nil {
		return fmt.Errorf("convert data: %w", err)
	}
	return s.store.DataPut(entry)
}

func buildEntryFromData(item *gophkeeper.DataInfo) (model.DataEntry, error) {
	discriminator, err := item.Discriminator()
	if err != nil {
		return model.DataEntry{}, fmt.Errorf("get discriminator: %w", err)
	}

	switch discriminator {
	case string(gophkeeper.DataTypeLOGINPASSWORD):
		data, err := item.AsLoginPasswordDataInfo()
		if err != nil {
			return model.DataEntry{}, err
		}
		payload := fmt.Sprintf(`{"login":%q,"password":%q}`, data.Login, data.Password)
		localID := uuid.New()
		etag := data.UpdatedAt.Format(time.RFC3339)

		return model.DataEntry{
			ID:         localID,
			RemoteID:   &data.Id,
			Type:       model.EntryTypeLoginPassword,
			Payload:    payload,
			SyncStatus: model.SyncStatusSynced,
			ServerEtag: &etag,
			CreatedAt:  data.CreatedAt,
			UpdatedAt:  data.UpdatedAt,
		}, nil

	case string(gophkeeper.DataTypeBANKCARD):
		data, err := item.AsBankCardDataInfo()
		if err != nil {
			return model.DataEntry{}, err
		}
		payload := fmt.Sprintf(`{"cardNumber":%q,"cardHolder":%q,"cardExpiry":%q}`, data.CardNumber, data.CardHolder, data.CardExpiry)
		localID := uuid.New()
		etag := data.UpdatedAt.Format(time.RFC3339)

		return model.DataEntry{
			ID:         localID,
			RemoteID:   &data.Id,
			Type:       model.EntryTypeBankCard,
			Payload:    payload,
			SyncStatus: model.SyncStatusSynced,
			ServerEtag: &etag,
			CreatedAt:  data.CreatedAt,
			UpdatedAt:  data.UpdatedAt,
		}, nil

	case string(gophkeeper.DataTypeTEXT):
		data, err := item.AsTextDataInfo()
		if err != nil {
			return model.DataEntry{}, err
		}
		payload := fmt.Sprintf(`{"text":%q}`, data.Text)
		localID := uuid.New()
		etag := data.UpdatedAt.Format(time.RFC3339)

		return model.DataEntry{
			ID:         localID,
			RemoteID:   &data.Id,
			Type:       model.EntryTypeText,
			Payload:    payload,
			SyncStatus: model.SyncStatusSynced,
			ServerEtag: &etag,
			CreatedAt:  data.CreatedAt,
			UpdatedAt:  data.UpdatedAt,
		}, nil

	case string(gophkeeper.DataTypeFILE):
		data, err := item.AsFileDataInfo()
		if err != nil {
			return model.DataEntry{}, err
		}
		fileIDs := "["
		for i, fid := range data.FileIds {
			if i > 0 {
				fileIDs += ","
			}
			fileIDs += fmt.Sprintf(`%q`, fid.String())
		}
		fileIDs += "]"
		payload := fmt.Sprintf(`{"fileIds":%s}`, fileIDs)
		localID := uuid.New()
		etag := data.UpdatedAt.Format(time.RFC3339)

		return model.DataEntry{
			ID:         localID,
			RemoteID:   &data.Id,
			Type:       model.EntryTypeFile,
			Payload:    payload,
			SyncStatus: model.SyncStatusSynced,
			ServerEtag: &etag,
			CreatedAt:  data.CreatedAt,
			UpdatedAt:  data.UpdatedAt,
		}, nil

	default:
		return model.DataEntry{}, fmt.Errorf("unknown data type: %s", discriminator)
	}
}

func buildDataFromEntry(entry model.DataEntry) (*gophkeeper.Data, error) {
	data := &gophkeeper.Data{}

	switch entry.Type {
	case model.EntryTypeLoginPassword:
		var lp gophkeeper.LoginPasswordData
		if err := json.Unmarshal([]byte(entry.Payload), &lp); err != nil {
			return nil, fmt.Errorf("parse login_password payload: %w", err)
		}
		lp.Type = gophkeeper.LoginPasswordDataTypeLOGINPASSWORD
		if err := data.FromLoginPasswordData(lp); err != nil {
			return nil, err
		}

	case model.EntryTypeBankCard:
		var bc gophkeeper.BankCardData
		if err := json.Unmarshal([]byte(entry.Payload), &bc); err != nil {
			return nil, fmt.Errorf("parse bank_card payload: %w", err)
		}
		bc.Type = gophkeeper.BankCardDataTypeBANKCARD
		if err := data.FromBankCardData(bc); err != nil {
			return nil, err
		}

	case model.EntryTypeText:
		var td gophkeeper.TextData
		if err := json.Unmarshal([]byte(entry.Payload), &td); err != nil {
			return nil, fmt.Errorf("parse text payload: %w", err)
		}
		td.Type = gophkeeper.TextDataTypeTEXT
		if err := data.FromTextData(td); err != nil {
			return nil, err
		}

	case model.EntryTypeFile:
		var fd gophkeeper.FileData
		if err := json.Unmarshal([]byte(entry.Payload), &fd); err != nil {
			return nil, fmt.Errorf("parse file payload: %w", err)
		}
		fd.Type = gophkeeper.FileDataTypeFILE
		if err := data.FromFileData(fd); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unknown data type: %s", entry.Type)
	}

	return data, nil
}

func extractDataID(item *gophkeeper.DataInfo) (uuid.UUID, error) {
	if item == nil {
		return uuid.Nil, fmt.Errorf("no data")
	}

	discriminator, err := item.Discriminator()
	if err != nil {
		return uuid.Nil, err
	}

	switch discriminator {
	case string(gophkeeper.DataTypeLOGINPASSWORD):
		if data, err := item.AsLoginPasswordDataInfo(); err == nil {
			return data.Id, nil
		}
	case string(gophkeeper.DataTypeBANKCARD):
		if data, err := item.AsBankCardDataInfo(); err == nil {
			return data.Id, nil
		}
	case string(gophkeeper.DataTypeTEXT):
		if data, err := item.AsTextDataInfo(); err == nil {
			return data.Id, nil
		}
	case string(gophkeeper.DataTypeFILE):
		if data, err := item.AsFileDataInfo(); err == nil {
			return data.Id, nil
		}
	}

	return uuid.Nil, fmt.Errorf("unknown data type: %s", discriminator)
}

func strPtr(s string) *string {
	return &s
}
