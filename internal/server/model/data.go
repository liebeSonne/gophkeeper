package model

import (
	"time"

	"github.com/google/uuid"
)

type DataType string

const (
	DataTypeLoginPassword DataType = "LOGIN_PASSWORD"
	DataTypeBankCard      DataType = "BANK_CARD"
	DataTypeText          DataType = "TEXT"
	DataTypeFile          DataType = "FILE"
)

type Data struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      DataType  `json:"type"`
	Payload   []byte    `json:"-"`        // encrypted JSON (AES-GCM)
	Metadata  string    `json:"metadata"` // unencrypted
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// nolint: gosec
type LoginPasswordPayload struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type BankCardPayload struct {
	CardNumber string `json:"card_number"`
	CardHolder string `json:"card_holder"`
	CardExpiry string `json:"card_expiry"`
	CardCVV    string `json:"card_cvv"`
}

type TextPayload struct {
	Text string `json:"text"`
}

type FilePayload struct {
	FileIDs []uuid.UUID `json:"file_ids"`
}
