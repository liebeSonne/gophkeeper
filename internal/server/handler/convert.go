package handler

import (
	server "github.com/liebeSonne/gophkeeper/api/swagger"
	"github.com/liebeSonne/gophkeeper/internal/model"
)

func convertTokenToAPI(token model.Token) server.TokenResponse {
	return server.TokenResponse{
		AccessToken:           token.AccessToken,
		ExpiresIn:             token.ExpiresIn,
		TokenType:             "Bearer",
		RefreshToken:          token.RefreshToken,
		RefreshTokenExpiresIn: token.RefreshTokenExpiresIn,
	}
}
