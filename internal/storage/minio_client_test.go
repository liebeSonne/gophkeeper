package storage

import (
	"testing"

	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinIOClient(t *testing.T) {
	testCases := []struct {
		name      string
		endpoint  string
		accessKey string
		secretKey string
		secure    bool
		wantErr   bool
	}{
		{
			name:      "valid endpoint",
			endpoint:  "localhost:9000",
			accessKey: "minioadmin",
			secretKey: "minioadmin",
			secure:    false,
			wantErr:   false,
		},
		{
			name:      "empty endpoint",
			endpoint:  "",
			accessKey: "minioadmin",
			secretKey: "minioadmin",
			secure:    false,
			wantErr:   true,
		},
		{
			name:      "empty access key",
			endpoint:  "localhost:9000",
			accessKey: "",
			secretKey: "minioadmin",
			secure:    false,
			wantErr:   false,
		},
		{
			name:      "empty secret key",
			endpoint:  "localhost:9000",
			accessKey: "minioadmin",
			secretKey: "",
			secure:    false,
			wantErr:   false,
		},
		{
			name:      "secure connection",
			endpoint:  "minio.example.com",
			accessKey: "minioadmin",
			secretKey: "minioadmin",
			secure:    true,
			wantErr:   false,
		},
		{
			name:      "invalid endpoint format",
			endpoint:  "not-a-valid-endpoint",
			accessKey: "minioadmin",
			secretKey: "minioadmin",
			secure:    false,
			wantErr:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logger intlogger.Logger
			client, err := NewMinIOClient(tc.endpoint, tc.accessKey, tc.secretKey, tc.secure, logger)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, client)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}
