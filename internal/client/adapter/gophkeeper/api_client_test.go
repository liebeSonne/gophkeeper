package gophkeeper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

func TestAPIClient(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		handler  func(http.ResponseWriter, *http.Request)
		testFunc func(*testing.T, *Client)
	}{
		{
			name: "health check success",
			path: "/api/v1/health",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"status":"ok"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				resp, err := c.HealthCheck(context.Background())
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "health check server error",
			path: "/api/v1/health",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(`{"message":"internal error"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				_, err := c.HealthCheck(context.Background())
				require.Error(t, err)
			},
		},
		{
			name: "register user success",
			path: "/api/v1/auth/register",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"access_token":"test","refresh_token":"test","expires_in":3600,"refresh_token_expires_in":86400,"token_type":"Bearer"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				resp, err := c.RegisterUser(context.Background(), "testuser", "testpass")
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "register user conflict",
			path: "/api/v1/auth/register",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, err := w.Write([]byte(`{"message":"user already exists"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				_, err := c.RegisterUser(context.Background(), "testuser", "testpass")
				require.Error(t, err)
			},
		},
		{
			name: "login user success",
			path: "/api/v1/auth/login",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"access_token":"test","refresh_token":"test","expires_in":3600,"refresh_token_expires_in":86400,"token_type":"Bearer"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				resp, err := c.LoginUser(context.Background(), "testuser", "testpass")
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "login user invalid credentials",
			path: "/api/v1/auth/login",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, err := w.Write([]byte(`{"message":"invalid credentials"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				_, err := c.LoginUser(context.Background(), "testuser", "wrongpass")
				require.Error(t, err)
			},
		},
		{
			name: "refresh token success",
			path: "/api/v1/auth/refresh",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"access_token":"new-token","refresh_token":"new-refresh","expires_in":3600,"refresh_token_expires_in":86400,"token_type":"Bearer"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				resp, err := c.RefreshTokenAPI(context.Background(), "test-refresh-token")
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "create data success",
			path: "/api/v1/data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, err := w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000","type":"TEXT","text":"test","metadata":"test metadata","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				data := gophkeeper.Data{}
				_ = data.FromTextData(gophkeeper.TextData{
					Type:     gophkeeper.TextDataTypeTEXT,
					Text:     "test",
					Metadata: strPtr("test metadata"),
				})
				resp, err := c.CreateData(context.Background(), data)
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "list data success",
			path: "/api/v1/data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"items":[],"page":1,"page_size":20,"total":0,"total_pages":1}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				resp, err := c.ListData(context.Background(), nil, nil, nil, nil)
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "delete data success",
			path: "/api/v1/data/550e8400-e29b-41d4-a716-446655440000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			testFunc: func(_ *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				err := c.DeleteData(context.Background(), [16]byte{})
				assert.NoError(t, err)
			},
		},
		{
			name: "init file upload success",
			path: "/api/v1/files/upload",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, err := w.Write([]byte(`{"file_id":"550e8400-e29b-41d4-a716-446655440000","chunks_count":2}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				resp, err := c.InitUpload(context.Background(), gophkeeper.FileInitRequest{
					Name:        "testfile.txt",
					MimeType:    "text/plain",
					Size:        1024,
					ChunksCount: 2,
				})
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "list files success",
			path: "/api/v1/files",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"items":[],"page":1,"page_size":20,"total":0,"total_pages":1}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				resp, err := c.ListFiles(context.Background(), nil, nil, nil)
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "delete file success",
			path: "/api/v1/files/550e8400-e29b-41d4-a716-446655440000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			testFunc: func(_ *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				err := c.DeleteFile(context.Background(), [16]byte{})
				assert.NoError(t, err)
			},
		},
		{
			name: "get data success",
			path: "/api/v1/data/550e8400-e29b-41d4-a716-446655440000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000","type":"TEXT","text":"test","metadata":"test metadata","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				resp, err := c.GetData(context.Background(), openapi_types.UUID([16]byte{}))
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "get data not found",
			path: "/api/v1/data/nonexistent",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"message":"data not found"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				_, err := c.GetData(context.Background(), openapi_types.UUID([16]byte{}))
				require.Error(t, err)
			},
		},
		{
			name: "update data success",
			path: "/api/v1/data/550e8400-e29b-41d4-a716-446655440000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"id":"550e8400-e29b-41d4-a716-446655440000","type":"TEXT","text":"updated","metadata":"updated metadata","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				data := gophkeeper.Data{}
				_ = data.FromTextData(gophkeeper.TextData{
					Type:     gophkeeper.TextDataTypeTEXT,
					Text:     "updated",
					Metadata: strPtr("updated metadata"),
				})
				resp, err := c.UpdateData(context.Background(), openapi_types.UUID([16]byte{}), data)
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "set auth token adds header",
			path: "/api/v1/data",
			handler: func(w http.ResponseWriter, r *http.Request) {
				auth := r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				// nolint:gosec
				_, err := w.Write([]byte(`{"auth":"` + auth + `"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("my-test-token")
				resp, err := c.ListData(context.Background(), nil, nil, nil, nil)
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "refresh token invalid",
			path: "/api/v1/auth/refresh",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, err := w.Write([]byte(`{"message":"invalid refresh token"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				_, err := c.RefreshTokenAPI(context.Background(), "invalid-token")
				require.Error(t, err)
			},
		},
		{
			name: "create data unauthorized",
			path: "/api/v1/data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, err := w.Write([]byte(`{"message":"unauthorized"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				data := gophkeeper.Data{}
				_ = data.FromTextData(gophkeeper.TextData{
					Type: gophkeeper.TextDataTypeTEXT,
					Text: "test",
				})
				_, err := c.CreateData(context.Background(), data)
				require.Error(t, err)
			},
		},
		{
			name: "list files unauthorized",
			path: "/api/v1/files",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, err := w.Write([]byte(`{"message":"unauthorized"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				_, err := c.ListFiles(context.Background(), nil, nil, nil)
				require.Error(t, err)
			},
		},
		{
			name: "complete upload success",
			path: "/api/v1/files/upload/complete",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"file_id":"550e8400-e29b-41d4-a716-446655440000","name":"testfile.txt","status":"COMPLETED"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				resp, err := c.CompleteUpload(context.Background(), openapi_types.UUID([16]byte{}))
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "complete upload not found",
			path: "/api/v1/files/upload/complete",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"message":"file not found"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				_, err := c.CompleteUpload(context.Background(), openapi_types.UUID([16]byte{}))
				require.Error(t, err)
			},
		},
		{
			name: "download file success",
			path: "/api/v1/files/550e8400-e29b-41d4-a716-446655440000/download",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("file content"))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				resp, err := c.DownloadFile(context.Background(), openapi_types.UUID([16]byte{}))
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name: "download file not found",
			path: "/api/v1/files/nonexistent/download",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"message":"file not found"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				_, err := c.DownloadFile(context.Background(), openapi_types.UUID([16]byte{}))
				require.Error(t, err)
			},
		},
		{
			name: "set refresh token",
			path: "/api/v1/data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"items":[],"page":1,"page_size":20,"total":0,"total_pages":1}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetRefreshToken(func(_ context.Context) error {
					return nil
				})
				c.SetAuthToken("test-token")
				_, err := c.ListData(context.Background(), nil, nil, nil, nil)
				require.NoError(t, err)
			},
		},
		{
			name: "api error returns error",
			path: "/api/v1/data",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte(`{"message":"test error message"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				_, err := c.ListData(context.Background(), nil, nil, nil, nil)
				require.Error(t, err)
			},
		},
		{
			name: "delete file success",
			path: "/api/v1/files/550e8400-e29b-41d4-a716-446655440000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			testFunc: func(_ *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				err := c.DeleteFile(context.Background(), openapi_types.UUID([16]byte{}))
				assert.NoError(t, err)
			},
		},
		{
			name: "delete file not found",
			path: "/api/v1/files/nonexistent",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`{"message":"file not found"}`))
				assert.NoError(t, err)
			},
			testFunc: func(t *testing.T, c *Client) {
				c.SetAuthToken("test-token")
				err := c.DeleteFile(context.Background(), openapi_types.UUID([16]byte{}))
				require.Error(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.handler))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			tc.testFunc(t, client)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
