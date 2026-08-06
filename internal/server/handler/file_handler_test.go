// nolint:goconst
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	apperrors "github.com/liebeSonne/gophkeeper/internal/errors"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	authctx "github.com/liebeSonne/gophkeeper/internal/server/auth"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
)

func TestFileHandlers(t *testing.T) {
	userID := uuid.New()
	fileID := uuid.New()

	testCases := []struct {
		name             string
		handlerFn        func(server.ServerInterface, http.ResponseWriter, *http.Request)
		path             string
		body             interface{}
		withUser         bool
		setupMock        func(*MockFileService)
		expectedStatus   int
		assertResponseFn func(*httptest.ResponseRecorder)
	}{
		{
			name:      "init upload success",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.InitUpload(w, r) },
			path:      "/api/v1/files/upload",
			withUser:  true,
			body: server.FileInitRequest{
				Name:        "testfile.txt",
				MimeType:    "text/plain",
				Size:        1024,
				ChunksCount: 2,
			},
			setupMock: func(m *MockFileService) {
				m.On("InitUpload", mock.Anything, userID, "testfile.txt", "text/plain", int64(1024), 2).
					Return(model.File{ID: fileID, UserID: userID, Name: "testfile.txt"}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:      "init upload missing name",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.InitUpload(w, r) },
			path:      "/api/v1/files/upload",
			withUser:  true,
			body: map[string]interface{}{
				"mime_type":    "text/plain",
				"size":         float64(1024),
				"chunks_count": float64(2),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "init upload unauthorized",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.InitUpload(w, r) },
			path:      "/api/v1/files/upload",
			withUser:  false,
			body: server.FileInitRequest{
				Name:        "testfile.txt",
				MimeType:    "text/plain",
				Size:        1024,
				ChunksCount: 2,
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "complete upload success",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.CompleteUpload(w, r) },
			path:      "/api/v1/files/upload/complete",
			withUser:  true,
			body: server.FileCompleteRequest{
				FileId: fileID,
			},
			setupMock: func(m *MockFileService) {
				m.On("CompleteUpload", mock.Anything, fileID, userID).
					Return(model.File{ID: fileID, UserID: userID, Status: model.FileStatusCompleted}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "complete upload unauthorized",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.CompleteUpload(w, r) },
			path:      "/api/v1/files/upload/complete",
			withUser:  false,
			body: server.FileCompleteRequest{
				FileId: fileID,
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "download file success",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.DownloadFile(w, r, fileID)
			},
			path:     "/api/v1/files/" + fileID.String() + "/download",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("DownloadFile", mock.Anything, fileID, userID).
					Return(&readCloser{bytes.NewReader([]byte("file content"))}, "text/plain", int64(12), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "download file not found",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.DownloadFile(w, r, fileID)
			},
			path:     "/api/v1/files/" + fileID.String() + "/download",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("DownloadFile", mock.Anything, fileID, userID).
					Return(nil, "", int64(0), apperrors.ErrFileNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "download file unauthorized",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.DownloadFile(w, r, fileID)
			},
			path:           "/api/v1/files/" + fileID.String() + "/download",
			withUser:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "delete file success",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.DeleteFile(w, r, fileID)
			},
			path:     "/api/v1/files/" + fileID.String(),
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("DeleteFile", mock.Anything, fileID, userID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "delete file not found",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.DeleteFile(w, r, fileID)
			},
			path:     "/api/v1/files/" + fileID.String(),
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("DeleteFile", mock.Anything, fileID, userID).Return(apperrors.ErrFileNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "delete file unauthorized",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.DeleteFile(w, r, fileID)
			},
			path:           "/api/v1/files/" + fileID.String(),
			withUser:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "list files success",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.ListFiles(w, r, server.ListFilesParams{})
			},
			path:     "/api/v1/files",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("ListFiles", mock.Anything, userID, 1, 20, (*string)(nil)).
					Return([]model.File{
						{ID: uuid.New(), Name: "test.txt", Size: 1024},
						{ID: uuid.New(), Name: "test2.txt", Size: 2048},
					}, 2, nil)
			},
			expectedStatus: http.StatusOK,
			assertResponseFn: func(rr *httptest.ResponseRecorder) {
				var resp server.FilesList
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Len(t, resp.Items, 2)
				assert.Equal(t, 2, resp.Total)
			},
		},
		{
			name: "list files with pagination",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.ListFiles(w, r, server.ListFilesParams{Page: intPtr(2), PageSize: intPtr(10)})
			},
			path:     "/api/v1/files?page=2&page_size=10",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("ListFiles", mock.Anything, userID, 2, 10, (*string)(nil)).
					Return([]model.File{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "list files with search",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				query := "test"
				h.ListFiles(w, r, server.ListFilesParams{Query: &query})
			},
			path:     "/api/v1/files?query=test",
			withUser: true,
			setupMock: func(m *MockFileService) {
				query := "test"
				m.On("ListFiles", mock.Anything, userID, 1, 20, &query).
					Return([]model.File{
						{ID: uuid.New(), Name: "test.txt", Size: 1024},
					}, 1, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "list files unauthorized",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.ListFiles(w, r, server.ListFilesParams{})
			},
			path:           "/api/v1/files",
			withUser:       false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "list files with service error",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) {
				h.ListFiles(w, r, server.ListFilesParams{})
			},
			path:     "/api/v1/files",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("ListFiles", mock.Anything, userID, 1, 20, (*string)(nil)).
					Return(nil, 0, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "complete upload file not found",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.CompleteUpload(w, r) },
			path:      "/api/v1/files/upload/complete",
			withUser:  true,
			body: server.FileCompleteRequest{
				FileId: fileID,
			},
			setupMock: func(m *MockFileService) {
				m.On("CompleteUpload", mock.Anything, fileID, userID).
					Return(model.File{}, apperrors.ErrFileNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "complete upload access denied",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.CompleteUpload(w, r) },
			path:      "/api/v1/files/upload/complete",
			withUser:  true,
			body: server.FileCompleteRequest{
				FileId: fileID,
			},
			setupMock: func(m *MockFileService) {
				m.On("CompleteUpload", mock.Anything, fileID, userID).
					Return(model.File{}, apperrors.ErrFileAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:      "download file access denied",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.DownloadFile(w, r, fileID) },
			path:      "/api/v1/files/" + fileID.String() + "/download",
			withUser:  true,
			setupMock: func(m *MockFileService) {
				m.On("DownloadFile", mock.Anything, fileID, userID).
					Return(nil, "", int64(0), apperrors.ErrFileAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:      "delete file access denied",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.DeleteFile(w, r, fileID) },
			path:      "/api/v1/files/" + fileID.String(),
			withUser:  true,
			setupMock: func(m *MockFileService) {
				m.On("DeleteFile", mock.Anything, fileID, userID).Return(apperrors.ErrFileAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:      "init upload service error",
			handlerFn: func(h server.ServerInterface, w http.ResponseWriter, r *http.Request) { h.InitUpload(w, r) },
			path:      "/api/v1/files/upload",
			withUser:  true,
			body: server.FileInitRequest{
				Name:        "testfile.txt",
				MimeType:    "text/plain",
				Size:        1024,
				ChunksCount: 2,
			},
			setupMock: func(m *MockFileService) {
				m.On("InitUpload", mock.Anything, userID, "testfile.txt", "text/plain", int64(1024), 2).
					Return(model.File{}, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			mockFileService := NewMockFileService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockFileService)
			}
			l := intlogger.NewMockLogger(t)
			l.On("Error", mock.Anything, mock.Anything).Return().Maybe()

			h := NewServerHandler(mockAuthService, mockDataService, mockFileService, l)

			var bodyBytes []byte
			if tc.body != nil {
				jsonData, err := json.Marshal(tc.body)
				require.NoError(t, err)
				bodyBytes = jsonData
			} else {
				bodyBytes = []byte("{}")
			}

			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(bodyBytes))
			if tc.withUser {
				ctx := authctx.CreateTokenContext(req.Context(), userID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			tc.handlerFn(h, rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "response body: %s", rr.Body.String())

			if tc.assertResponseFn != nil {
				tc.assertResponseFn(rr)
			}

			mockFileService.AssertExpectations(t)
		})
	}
}

type readCloser struct {
	*bytes.Reader
}

func (r *readCloser) Close() error {
	return nil
}

func intPtr(i int) *int {
	return &i
}

func TestUploadChunk(t *testing.T) {
	userID := uuid.New()
	fileID := uuid.New()

	testCases := []struct {
		name           string
		withUser       bool
		setupMock      func(*MockFileService)
		fileID         string
		chunkIndex     string
		data           string
		expectedStatus int
	}{
		{
			name:     "upload chunk success",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("UploadChunk", mock.Anything, fileID, userID, 0, mock.AnythingOfType("[]uint8")).Return(nil)
			},
			fileID:         fileID.String(),
			chunkIndex:     "0",
			data:           "chunk data content",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "upload chunk unauthorized",
			withUser:       false,
			fileID:         fileID.String(),
			chunkIndex:     "0",
			data:           "chunk data content",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "upload chunk file not found",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("UploadChunk", mock.Anything, fileID, userID, 0, mock.AnythingOfType("[]uint8")).Return(apperrors.ErrFileNotFound)
			},
			fileID:         fileID.String(),
			chunkIndex:     "0",
			data:           "chunk data content",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "upload chunk access denied",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("UploadChunk", mock.Anything, fileID, userID, 0, mock.AnythingOfType("[]uint8")).Return(apperrors.ErrFileAccessDenied)
			},
			fileID:         fileID.String(),
			chunkIndex:     "0",
			data:           "chunk data content",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "upload chunk invalid file id",
			withUser:       true,
			fileID:         "invalid-uuid",
			chunkIndex:     "0",
			data:           "chunk data content",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "upload chunk invalid chunk index",
			withUser:       true,
			fileID:         fileID.String(),
			chunkIndex:     "invalid",
			data:           "chunk data content",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "upload chunk service error",
			withUser: true,
			setupMock: func(m *MockFileService) {
				m.On("UploadChunk", mock.Anything, fileID, userID, 0, mock.AnythingOfType("[]uint8")).Return(errors.New("storage error"))
			},
			fileID:         fileID.String(),
			chunkIndex:     "0",
			data:           "chunk data content",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockAuthService := NewMockAuthService(t)
			mockDataService := NewMockDataService(t)
			mockFileService := NewMockFileService(t)
			if tc.setupMock != nil {
				tc.setupMock(mockFileService)
			}
			l := intlogger.NewMockLogger(t)
			l.On("Error", mock.Anything, mock.Anything).Return().Maybe()

			h := NewServerHandler(mockAuthService, mockDataService, mockFileService, l)

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			_ = writer.WriteField("file_id", tc.fileID)
			_ = writer.WriteField("chunk_index", tc.chunkIndex)

			dataWriter, err := writer.CreateFormFile("data", "chunk.bin")
			require.NoError(t, err)
			_, _ = dataWriter.Write([]byte(tc.data))
			_ = writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload/"+fileID.String()+"/chunk/0", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			if tc.withUser {
				ctx := authctx.CreateTokenContext(req.Context(), userID)
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			h.UploadChunk(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "response body: %s", rr.Body.String())
			mockFileService.AssertExpectations(t)
		})
	}
}
