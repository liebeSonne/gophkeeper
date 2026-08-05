package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
)

func TestConvertDataTypeFromAPI(t *testing.T) {
	testCases := []struct {
		name      string
		apiType   server.DataType
		wantType  model.DataType
		wantError bool
	}{
		{
			name:     "login_password",
			apiType:  server.DataTypeLOGINPASSWORD,
			wantType: model.DataTypeLoginPassword,
		},
		{
			name:     "bank_card",
			apiType:  server.DataTypeBANKCARD,
			wantType: model.DataTypeBankCard,
		},
		{
			name:     "text",
			apiType:  server.DataTypeTEXT,
			wantType: model.DataTypeText,
		},
		{
			name:     "file",
			apiType:  server.DataTypeFILE,
			wantType: model.DataTypeFile,
		},
		{
			name:      "unknown",
			apiType:   "UNKNOWN",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, err := convertDataTypeFromAPI(tc.apiType)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantType, gotType)
			}
		})
	}
}

func TestConvertDataToDataInfo(t *testing.T) {
	dataID := uuid.New()
	now := time.Now()

	testCases := []struct {
		name      string
		data      model.Data
		wantError bool
	}{
		{
			name: "login_password",
			data: model.Data{
				ID:        dataID,
				UserID:    uuid.New(),
				Type:      model.DataTypeLoginPassword,
				Payload:   []byte(`{"login":"user","password":"pass"}`),
				Metadata:  "test metadata",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "bank_card",
			data: model.Data{
				ID:        dataID,
				UserID:    uuid.New(),
				Type:      model.DataTypeBankCard,
				Payload:   []byte(`{"card_number":"4111","card_holder":"John","card_expiry":"12/25","card_cvv":"123"}`),
				Metadata:  "test metadata",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "text",
			data: model.Data{
				ID:        dataID,
				UserID:    uuid.New(),
				Type:      model.DataTypeText,
				Payload:   []byte(`{"text":"some text"}`),
				Metadata:  "test metadata",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "file",
			data: model.Data{
				ID:        dataID,
				UserID:    uuid.New(),
				Type:      model.DataTypeFile,
				Payload:   []byte(`{"file_ids":["550e8400-e29b-41d4-a716-446655440000"]}`),
				Metadata:  "test metadata",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "empty payload",
			data: model.Data{
				ID:        dataID,
				UserID:    uuid.New(),
				Type:      model.DataTypeText,
				Payload:   []byte{},
				Metadata:  "test metadata",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantError: true,
		},
		{
			name: "invalid json",
			data: model.Data{
				ID:        dataID,
				UserID:    uuid.New(),
				Type:      model.DataTypeText,
				Payload:   []byte(`{invalid}`),
				Metadata:  "test metadata",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantError: true,
		},
		{
			name: "unknown type",
			data: model.Data{
				ID:        dataID,
				UserID:    uuid.New(),
				Type:      model.DataType(99),
				Payload:   []byte(`{}`),
				Metadata:  "test metadata",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := convertDataToDataInfo(tc.data)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, info)
			}
		})
	}
}

func TestConvertFileStatusToAPI(t *testing.T) {
	testCases := []struct {
		name      string
		status    model.FileStatus
		want      server.FileStatus
		wantError bool
	}{
		{
			name:   "in_progress",
			status: model.FileStatusInProgress,
			want:   server.INPROGRESS,
		},
		{
			name:   "completed",
			status: model.FileStatusCompleted,
			want:   server.COMPLETED,
		},
		{
			name:   "failed",
			status: model.FileStatusFailed,
			want:   server.FAILED,
		},
		{
			name:      "unknown",
			status:    model.FileStatus(99),
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertFileStatusToAPI(tc.status)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestConvertFileToAPI(t *testing.T) {
	fileID := uuid.New()
	now := time.Now()

	file := model.File{
		ID:          fileID,
		UserID:      uuid.New(),
		Name:        "test.txt",
		MimeType:    "text/plain",
		Size:        1024,
		ChunksCount: 2,
		Status:      model.FileStatusCompleted,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	info, err := convertFileToAPI(file)
	require.NoError(t, err)
	assert.Equal(t, fileID, info.Id)
	assert.Equal(t, "test.txt", info.Name)
	assert.Equal(t, "text/plain", info.MimeType)
	assert.Equal(t, int64(1024), info.Size)
	assert.Equal(t, 2, info.ChunksCount)
	assert.Equal(t, server.COMPLETED, info.Status)
}

func TestConvertDataMetadataFromAPIData(t *testing.T) {
	testCases := []struct {
		name     string
		data     *server.Data
		wantMeta string
	}{
		{
			name: "login_password with metadata",
			data: func() *server.Data {
				d := &server.Data{}
				meta := "test metadata"
				_ = d.FromLoginPasswordData(server.LoginPasswordData{
					Type:     server.LoginPasswordDataTypeLOGINPASSWORD,
					Login:    "user",
					Password: "pass",
					Metadata: &meta,
				})
				return d
			}(),
			wantMeta: "test metadata",
		},
		{
			name: "text without metadata",
			data: func() *server.Data {
				d := &server.Data{}
				_ = d.FromTextData(server.TextData{
					Type: server.TextDataTypeTEXT,
					Text: "some text",
				})
				return d
			}(),
			wantMeta: "",
		},
		{
			name: "bank_card with metadata",
			data: func() *server.Data {
				d := &server.Data{}
				meta := "card metadata"
				_ = d.FromBankCardData(server.BankCardData{
					Type:       server.BankCardDataTypeBANKCARD,
					CardNumber: "4111",
					Metadata:   &meta,
				})
				return d
			}(),
			wantMeta: "card metadata",
		},
		{
			name: "file with metadata",
			data: func() *server.Data {
				d := &server.Data{}
				meta := "file metadata"
				_ = d.FromFileData(server.FileData{
					Type:     server.FileDataTypeFILE,
					Metadata: &meta,
				})
				return d
			}(),
			wantMeta: "file metadata",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotMeta := convertDataMetadataFromAPIData(tc.data)
			assert.Equal(t, tc.wantMeta, gotMeta)
		})
	}
}

func TestConvertDataTypeFromAPIData(t *testing.T) {
	testCases := []struct {
		name      string
		data      *server.Data
		wantType  model.DataType
		wantError bool
	}{
		{
			name: "login_password",
			data: func() *server.Data {
				d := &server.Data{}
				_ = d.FromLoginPasswordData(server.LoginPasswordData{
					Type: server.LoginPasswordDataTypeLOGINPASSWORD,
				})
				return d
			}(),
			wantType: model.DataTypeLoginPassword,
		},
		{
			name: "bank_card",
			data: func() *server.Data {
				d := &server.Data{}
				_ = d.FromBankCardData(server.BankCardData{
					Type: server.BankCardDataTypeBANKCARD,
				})
				return d
			}(),
			wantType: model.DataTypeBankCard,
		},
		{
			name: "text",
			data: func() *server.Data {
				d := &server.Data{}
				_ = d.FromTextData(server.TextData{
					Type: server.TextDataTypeTEXT,
				})
				return d
			}(),
			wantType: model.DataTypeText,
		},
		{
			name: "file",
			data: func() *server.Data {
				d := &server.Data{}
				_ = d.FromFileData(server.FileData{
					Type: server.FileDataTypeFILE,
				})
				return d
			}(),
			wantType: model.DataTypeFile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, err := convertDataTypeFromAPIData(tc.data)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantType, gotType)
			}
		})
	}
}

// nolint: goconst
func TestStrToPtr(t *testing.T) {
	s := "test"
	ptr := strToPtr(s)
	assert.NotNil(t, ptr)
	assert.Equal(t, s, *ptr)
}
