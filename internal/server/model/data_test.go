package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestData(t *testing.T) {
	testCases := []struct {
		name     string
		dataType DataType
		want     string
	}{
		{name: "login_password", dataType: DataTypeLoginPassword, want: "LOGIN_PASSWORD"},
		{name: "bank_card", dataType: DataTypeBankCard, want: "BANK_CARD"},
		{name: "text", dataType: DataTypeText, want: "TEXT"},
		{name: "file", dataType: DataTypeFile, want: "FILE"},
		{name: "unknown", dataType: DataType(99), want: "unknown_data_type(99)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.dataType.String())
		})
	}
}
