package model

// nolint: gosec
type Token struct {
	AccessToken           string
	ExpiresIn             int
	RefreshToken          string
	RefreshTokenExpiresIn int
}
