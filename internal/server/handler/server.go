package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/repository"
)

type serverHandler struct {
	authService AuthService
	logger      intlogger.Logger
}

func NewServerHandler(
	authService AuthService,
	logger intlogger.Logger,
) server.ServerInterface {
	return &serverHandler{
		authService: authService,
		logger:      logger,
	}
}

func (h *serverHandler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	err := enc.Encode(server.HealthResponse{
		Status: "ok",
	})
	if err != nil {
		h.logger.Error("error on encode response", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

func (h *serverHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req server.RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Login == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	token, err := h.authService.Register(ctx, req.Login, req.Password)
	if err != nil {
		if repository.IsConflict(err) {
			h.writeError(w, http.StatusConflict, "login already exists")
			return
		}
		h.logger.Error("error on register user", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	resp := convertTokenToAPI(token)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)
	err = enc.Encode(resp)
	if err != nil {
		h.logger.Error("error on encode response", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

func (h *serverHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req server.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Login == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	token, err := h.authService.Login(ctx, req.Login, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.logger.Error("error on login user", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	resp := convertTokenToAPI(token)

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err = enc.Encode(resp)
	if err != nil {
		h.logger.Error("error on encode response", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

func (h *serverHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req server.RefreshRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		h.writeError(w, http.StatusBadRequest, "refresh token is required")
		return
	}

	token, err := h.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.writeError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		h.logger.Error("error on refresh token", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	resp := convertTokenToAPI(token)

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err = enc.Encode(resp)
	if err != nil {
		h.logger.Error("error on encode response", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

func (h *serverHandler) CreateData(w http.ResponseWriter, r *http.Request) {
	_ = r
	h.writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *serverHandler) ListData(w http.ResponseWriter, r *http.Request, params server.ListDataParams) {
	_ = r
	_ = params
	h.writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *serverHandler) DeleteData(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	_ = r
	_ = id
	h.writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *serverHandler) GetData(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	_ = r
	_ = id
	h.writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *serverHandler) UpdateData(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	_ = r
	_ = id
	h.writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *serverHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	err := enc.Encode(server.Error{Message: message})
	if err != nil {
		h.logger.Error("error on encode error message")
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}
