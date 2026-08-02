package auth

import (
	"net/http"
	"strings"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	"github.com/liebeSonne/gophkeeper/internal/auth"
)

type Middleware struct {
	service Service
}

func NewAuthMiddleware(service Service) *Middleware {
	return &Middleware{
		service: service,
	}
}

func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, `{"message":"missing token"}`, http.StatusUnauthorized)
			return
		}

		userID, err := m.service.GetUserIDFromToken(token)
		if err != nil {
			http.Error(w, `{"message":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := auth.CreateTokenContext(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) ToMiddlewareFunc() server.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return m.Handle(next)
	}
}

func extractToken(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if token == "" {
		return ""
	}
	if strings.HasPrefix(token, "Bearer ") {
		return strings.TrimPrefix(token, "Bearer ")
	}
	return ""
}
