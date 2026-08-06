package model

import (
	"time"

	"github.com/google/uuid"
)

type SyncStatus string

const (
	SyncStatusSynced   SyncStatus = "synced"
	SyncStatusPending  SyncStatus = "pending"
	SyncStatusDeleting SyncStatus = "deleting"
	SyncStatusConflict SyncStatus = "conflict"
)

const (
	EntryTypeLoginPassword string = "LOGIN_PASSWORD"
	EntryTypeBankCard      string = "BANK_CARD"
	EntryTypeText          string = "TEXT"
	EntryTypeFile          string = "FILE"
)

type DataEntry struct {
	ID           uuid.UUID
	RemoteID     *uuid.UUID
	Type         string
	Payload      string
	SyncStatus   SyncStatus
	ServerEtag   *string
	LastSyncAt   *time.Time
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type DataFilter struct {
	Types  []string
	Query  string
	Limit  int
	Offset int
}
