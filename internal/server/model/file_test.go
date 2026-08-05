package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileStatusString(t *testing.T) {
	testCases := []struct {
		name   string
		status FileStatus
		want   string
	}{
		{name: "in_progress", status: FileStatusInProgress, want: "IN_PROGRESS"},
		{name: "completed", status: FileStatusCompleted, want: "COMPLETED"},
		{name: "failed", status: FileStatusFailed, want: "FAILED"},
		{name: "unknown", status: FileStatus(99), want: "UNKNOWN"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.status.String())
		})
	}
}
