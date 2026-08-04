package model

import "time"

// nolint: gosec
type Token struct {
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt time.Time
	RefreshExpiresAt     time.Time
}
