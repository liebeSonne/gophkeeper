package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceErrors(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "err_invalid_credentials",
			err:  ErrInvalidCredentials,
			want: "invalid credentials",
		},
		{
			name: "err_data_not_found",
			err:  ErrDataNotFound,
			want: "data not found",
		},
		{
			name: "err_data_access_denied",
			err:  ErrDataAccessDenied,
			want: "data access denied",
		},
		{
			name: "err_file_not_found",
			err:  ErrFileNotFound,
			want: "file not found",
		},
		{
			name: "err_file_access_denied",
			err:  ErrFileAccessDenied,
			want: "file access denied",
		},
		{
			name: "err_file_reference_invalid",
			err:  ErrFileReferenceInvalid,
			want: "one or more file references are invalid",
		},
		{
			name: "err_invalid_chunk_index",
			err:  ErrInvalidChunkIndex,
			want: "invalid chunk index",
		},
		{
			name: "err_invalid_chunks_count",
			err:  ErrInvalidChinksCount,
			want: "invalid chunks count (must be at least 1)",
		},
		{
			name: "err_file_upload_not_in_progress",
			err:  ErrFileUploadNotInProgress,
			want: "file upload not in progress",
		},
		{
			name: "err_file_not_completed",
			err:  ErrFileNotCompleted,
			want: "file not completed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.err.Error())
		})
	}
}
