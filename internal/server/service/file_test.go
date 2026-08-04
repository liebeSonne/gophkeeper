package service

import (
	"bytes"
	"context"
	"errors"
	"github.com/liebeSonne/gophkeeper/internal/storage"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/liebeSonne/gophkeeper/internal/crypto"
	apperrors "github.com/liebeSonne/gophkeeper/internal/errors"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
	"github.com/liebeSonne/gophkeeper/internal/server/repository"
)

func makeTestFile(userID, id uuid.UUID) model.File {
	return model.File{
		ID:          id,
		UserID:      userID,
		Name:        "testfile.txt",
		MimeType:    "text/plain",
		Size:        1024,
		ChunksCount: 2,
		Status:      model.FileStatusInProgress,
	}
}

func TestFileService_InitUpload(t *testing.T) {
	testCases := []struct {
		name        string
		setupMocks  func(*MockFileRepository, *crypto.MockEncryptor, *storage.MockMinIOClient)
		userID      uuid.UUID
		nameParam   string
		mimeType    string
		size        int64
		chunksCount int
		expectError bool
		expectFile  bool
	}{
		{
			name: "successful init",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				fileID := uuid.New()
				repo.On("NextID", mock.Anything).Return(fileID).Once()
				for i := 0; i < 2; i++ {
					repo.On("NextID", mock.Anything).Return(uuid.New())
				}
				repo.On("StoreFile", mock.Anything, mock.AnythingOfType("model.File")).Return(nil)
				repo.On("StoreFileChunks", mock.Anything, mock.AnythingOfType("[]model.FileChunk")).Return(nil)
			},
			userID:      uuid.New(),
			nameParam:   "testfile.txt",
			mimeType:    "text/plain",
			size:        1024,
			chunksCount: 2,
			expectFile:  true,
		},
		{
			name: "invalid chunks count",
			setupMocks: func(_ *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
			},
			userID:      uuid.New(),
			nameParam:   "testfile.txt",
			mimeType:    "text/plain",
			size:        1024,
			chunksCount: 0,
			expectError: true,
		},
		{
			name: "store file fails",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				repo.On("NextID", mock.Anything).Return(uuid.New())
				repo.On("StoreFile", mock.Anything, mock.AnythingOfType("model.File")).Return(errors.New("db error"))
			},
			userID:      uuid.New(),
			nameParam:   "testfile.txt",
			mimeType:    "text/plain",
			size:        1024,
			chunksCount: 2,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockFileRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockMinIO := storage.NewMockMinIOClient(t)

			tc.setupMocks(mockRepo, mockEnc, mockMinIO)

			l := intlogger.NewMockLogger(t)
			l.EXPECT().Warn(mock.Anything, mock.Anything).Return().Maybe()

			svc := NewFileService(mockRepo, mockEnc, mockMinIO, "testbucket", l)

			file, err := svc.InitUpload(context.Background(), tc.userID, tc.nameParam, tc.mimeType, tc.size, tc.chunksCount)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tc.expectFile {
				assert.Equal(t, tc.nameParam, file.Name)
				assert.Equal(t, tc.mimeType, file.MimeType)
				assert.Equal(t, tc.size, file.Size)
				assert.Equal(t, tc.chunksCount, file.ChunksCount)
				assert.Equal(t, model.FileStatusInProgress, file.Status)
			}
		})
	}
}

func TestFileService_UploadChunk(t *testing.T) {
	testUser := uuid.New()
	testFileID := uuid.New()

	testCases := []struct {
		name        string
		setupMocks  func(*MockFileRepository, *crypto.MockEncryptor, *storage.MockMinIOClient)
		fileID      uuid.UUID
		userID      uuid.UUID
		chunkIndex  int
		data        []byte
		expectError bool
		errorIs     error
	}{
		{
			name: "successful upload",
			setupMocks: func(repo *MockFileRepository, enc *crypto.MockEncryptor, minio *storage.MockMinIOClient) {
				file := makeTestFile(testUser, testFileID)
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
				enc.On("Encrypt", []byte("chunk data")).Return([]byte("encrypted"), nil)
				minio.On("PutObject", mock.Anything, "testbucket", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				repo.On("UpdateChunkUploaded", mock.Anything, testFileID, 0, true).Return(nil)
			},
			fileID:     testFileID,
			userID:     testUser,
			chunkIndex: 0,
			data:       []byte("chunk data"),
		},
		{
			name: "file not found",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				repo.On("GetFileByID", mock.Anything, testFileID).Return(model.File{}, repository.ErrNotFound)
			},
			fileID:      testFileID,
			userID:      testUser,
			chunkIndex:  0,
			data:        []byte("chunk data"),
			expectError: true,
			errorIs:     apperrors.ErrFileNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				otherUser := uuid.New()
				file := makeTestFile(otherUser, testFileID)
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
			},
			fileID:      testFileID,
			userID:      testUser,
			chunkIndex:  0,
			data:        []byte("chunk data"),
			expectError: true,
			errorIs:     apperrors.ErrFileAccessDenied,
		},
		{
			name: "invalid chunk index",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				file := makeTestFile(testUser, testFileID)
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
			},
			fileID:      testFileID,
			userID:      testUser,
			chunkIndex:  5,
			data:        []byte("chunk data"),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockFileRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockMinIO := storage.NewMockMinIOClient(t)

			tc.setupMocks(mockRepo, mockEnc, mockMinIO)

			l := intlogger.NewMockLogger(t)
			l.EXPECT().Warn(mock.Anything, mock.Anything).Return().Maybe()

			svc := NewFileService(mockRepo, mockEnc, mockMinIO, "testbucket", l)

			err := svc.UploadChunk(context.Background(), tc.fileID, tc.userID, tc.chunkIndex, tc.data)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileService_CompleteUpload(t *testing.T) {
	testUser := uuid.New()
	testFileID := uuid.New()

	testCases := []struct {
		name        string
		setupMocks  func(*MockFileRepository, *crypto.MockEncryptor, *storage.MockMinIOClient)
		fileID      uuid.UUID
		userID      uuid.UUID
		expectError bool
		errorIs     error
	}{
		{
			name: "file not found",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				repo.On("GetFileByID", mock.Anything, testFileID).Return(model.File{}, repository.ErrNotFound)
			},
			fileID:      testFileID,
			userID:      testUser,
			expectError: true,
			errorIs:     apperrors.ErrFileNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				otherUser := uuid.New()
				file := makeTestFile(otherUser, testFileID)
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
			},
			fileID:      testFileID,
			userID:      testUser,
			expectError: true,
			errorIs:     apperrors.ErrFileAccessDenied,
		},
		{
			name: "not all chunks uploaded",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				file := makeTestFile(testUser, testFileID)
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
				repo.On("GetUploadedChunksCount", mock.Anything, testFileID).Return(1, nil)
			},
			fileID:      testFileID,
			userID:      testUser,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockFileRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockMinIO := storage.NewMockMinIOClient(t)

			tc.setupMocks(mockRepo, mockEnc, mockMinIO)

			l := intlogger.NewMockLogger(t)
			l.EXPECT().Warn(mock.Anything, mock.Anything).Return().Maybe()

			svc := NewFileService(mockRepo, mockEnc, mockMinIO, "testbucket", l)

			file, err := svc.CompleteUpload(context.Background(), tc.fileID, tc.userID)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, model.FileStatusCompleted, file.Status)
		})
	}
}

func TestFileService_DownloadFile(t *testing.T) {
	testUser := uuid.New()
	testFileID := uuid.New()

	testCases := []struct {
		name        string
		setupMocks  func(*MockFileRepository, *crypto.MockEncryptor, *storage.MockMinIOClient)
		fileID      uuid.UUID
		userID      uuid.UUID
		expectError bool
		errorIs     error
	}{
		{
			name: "file not found",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				repo.On("GetFileByID", mock.Anything, testFileID).Return(model.File{}, repository.ErrNotFound)
			},
			fileID:      testFileID,
			userID:      testUser,
			expectError: true,
			errorIs:     apperrors.ErrFileNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				otherUser := uuid.New()
				file := makeTestFile(otherUser, testFileID)
				file.Status = model.FileStatusCompleted
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
			},
			fileID:      testFileID,
			userID:      testUser,
			expectError: true,
			errorIs:     apperrors.ErrFileAccessDenied,
		},
		{
			name: "file not completed",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				file := makeTestFile(testUser, testFileID)
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
			},
			fileID:      testFileID,
			userID:      testUser,
			expectError: true,
		},
		{
			name: "successful download",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, minio *storage.MockMinIOClient) {
				file := makeTestFile(testUser, testFileID)
				file.Status = model.FileStatusCompleted
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
				minio.On("GetObject", mock.Anything, "testbucket", mock.Anything).Return(
					io.NopCloser(bytes.NewReader([]byte("file content"))), int64(12), nil,
				)
			},
			fileID: testFileID,
			userID: testUser,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockFileRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockMinIO := storage.NewMockMinIOClient(t)

			tc.setupMocks(mockRepo, mockEnc, mockMinIO)

			l := intlogger.NewMockLogger(t)
			l.EXPECT().Warn(mock.Anything, mock.Anything).Return().Maybe()

			svc := NewFileService(mockRepo, mockEnc, mockMinIO, "testbucket", l)

			reader, mimeType, size, err := svc.DownloadFile(context.Background(), tc.fileID, tc.userID)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, reader)
			defer reader.Close()
			assert.Equal(t, "text/plain", mimeType)
			assert.Equal(t, int64(12), size)
		})
	}
}

func TestFileService_DeleteFile(t *testing.T) {
	testUser := uuid.New()
	testFileID := uuid.New()

	testCases := []struct {
		name        string
		setupMocks  func(*MockFileRepository, *crypto.MockEncryptor, *storage.MockMinIOClient)
		fileID      uuid.UUID
		userID      uuid.UUID
		expectError bool
		errorIs     error
	}{
		{
			name: "file not found",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				repo.On("GetFileByID", mock.Anything, testFileID).Return(model.File{}, repository.ErrNotFound)
			},
			fileID:      testFileID,
			userID:      testUser,
			expectError: true,
			errorIs:     apperrors.ErrFileNotFound,
		},
		{
			name: "access denied",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, _ *storage.MockMinIOClient) {
				otherUser := uuid.New()
				file := makeTestFile(otherUser, testFileID)
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
			},
			fileID:      testFileID,
			userID:      testUser,
			expectError: true,
			errorIs:     apperrors.ErrFileAccessDenied,
		},
		{
			name: "successful delete",
			setupMocks: func(repo *MockFileRepository, _ *crypto.MockEncryptor, minio *storage.MockMinIOClient) {
				file := makeTestFile(testUser, testFileID)
				repo.On("GetFileByID", mock.Anything, testFileID).Return(file, nil)
				minio.On("DeleteObject", mock.Anything, "testbucket", mock.Anything).Return(nil)
				for i := 0; i < file.ChunksCount; i++ {
					minio.On("DeleteObject", mock.Anything, "testbucket", mock.Anything).Return(nil)
				}
				repo.On("DeleteFile", mock.Anything, testFileID).Return(nil)
			},
			fileID: testFileID,
			userID: testUser,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockFileRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockMinIO := storage.NewMockMinIOClient(t)

			tc.setupMocks(mockRepo, mockEnc, mockMinIO)

			l := intlogger.NewMockLogger(t)
			l.EXPECT().Warn(mock.Anything, mock.Anything).Return().Maybe()

			svc := NewFileService(mockRepo, mockEnc, mockMinIO, "testbucket", l)

			err := svc.DeleteFile(context.Background(), tc.fileID, tc.userID)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFileService_ListFiles(t *testing.T) {
	testUser := uuid.New()

	testCases := []struct {
		name        string
		setupMocks  func(*MockFileRepository)
		page        int
		pageSize    int
		query       *string
		expectError bool
		expectCount int
		expectTotal int
	}{
		{
			name: "successful list",
			setupMocks: func(repo *MockFileRepository) {
				repo.On("CountByUserID", mock.Anything, testUser, (*string)(nil)).Return(3, nil)
				repo.On("ListByUserID", mock.Anything, testUser, (*string)(nil), mock.Anything, mock.Anything).Return([]model.File{
					makeTestFile(testUser, uuid.New()),
					makeTestFile(testUser, uuid.New()),
					makeTestFile(testUser, uuid.New()),
				}, nil)
			},
			page:        1,
			pageSize:    20,
			expectCount: 3,
			expectTotal: 3,
		},
		{
			name: "empty list",
			setupMocks: func(repo *MockFileRepository) {
				repo.On("CountByUserID", mock.Anything, testUser, (*string)(nil)).Return(0, nil)
				repo.On("ListByUserID", mock.Anything, testUser, (*string)(nil), mock.Anything, mock.Anything).Return([]model.File{}, nil)
			},
			page:        1,
			pageSize:    20,
			expectCount: 0,
			expectTotal: 0,
		},
		{
			name: "with query",
			setupMocks: func(repo *MockFileRepository) {
				repo.On("CountByUserID", mock.Anything, testUser, mock.AnythingOfType("*string")).Return(1, nil)
				repo.On("ListByUserID", mock.Anything, testUser, mock.AnythingOfType("*string"), mock.Anything, mock.Anything).Return([]model.File{
					makeTestFile(testUser, uuid.New()),
				}, nil)
			},
			page:        1,
			pageSize:    20,
			query:       strPtr("test"),
			expectCount: 1,
			expectTotal: 1,
		},
		{
			name: "count fails",
			setupMocks: func(repo *MockFileRepository) {
				repo.On("CountByUserID", mock.Anything, testUser, (*string)(nil)).Return(0, errors.New("db error"))
			},
			page:        1,
			pageSize:    20,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := NewMockFileRepository(t)
			mockEnc := crypto.NewMockEncryptor(t)
			mockMinIO := storage.NewMockMinIOClient(t)

			tc.setupMocks(mockRepo)

			l := intlogger.NewMockLogger(t)
			l.EXPECT().Warn(mock.Anything, mock.Anything).Return().Maybe()

			svc := NewFileService(mockRepo, mockEnc, mockMinIO, "testbucket", l)

			files, total, err := svc.ListFiles(context.Background(), testUser, tc.page, tc.pageSize, tc.query)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectTotal, total)
			assert.Len(t, files, tc.expectCount)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
