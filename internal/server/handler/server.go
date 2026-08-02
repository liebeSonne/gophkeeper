package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	authctx "github.com/liebeSonne/gophkeeper/internal/auth"
	apperrors "github.com/liebeSonne/gophkeeper/internal/errors"
	intlogger "github.com/liebeSonne/gophkeeper/internal/logger"
	"github.com/liebeSonne/gophkeeper/internal/model"
	"github.com/liebeSonne/gophkeeper/internal/repository"
)

const defaultPageSize = 20

type serverHandler struct {
	authService AuthService
	dataService DataService
	logger      intlogger.Logger
}

func NewServerHandler(
	authService AuthService,
	dataService DataService,
	logger intlogger.Logger,
) server.ServerInterface {
	return &serverHandler{
		authService: authService,
		dataService: dataService,
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
		if errors.Is(err, apperrors.ErrInvalidCredentials) {
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
		if errors.Is(err, apperrors.ErrInvalidCredentials) {
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
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req server.CreateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Data == nil {
		h.writeError(w, http.StatusBadRequest, "data is required")
		return
	}

	discriminator, err := req.Data.Discriminator()
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data type")
		return
	}
	dataType, err := convertDataTypeFromAPI(server.DataType(discriminator))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data type: "+discriminator)
		return
	}

	payloadJSON, err := req.Data.MarshalJSON()
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data payload")
		return
	}

	var metadata string
	val, valErr := req.Data.ValueByDiscriminator()
	if valErr == nil {
		switch d := val.(type) {
		case server.LoginPasswordData:
			if d.Metadata != nil {
				metadata = *d.Metadata
			}
		case server.BankCardData:
			if d.Metadata != nil {
				metadata = *d.Metadata
			}
		case server.TextData:
			if d.Metadata != nil {
				metadata = *d.Metadata
			}
		}
	}

	result, err := h.dataService.CreateData(ctx, userID, payloadJSON, dataType, metadata)
	if err != nil {
		h.logger.Error("error on create data", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	dataInfo, err := convertDataToDataInfo(result)
	if err != nil {
		h.logger.Error("error on convert data to info", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(server.CreateDataResponse{Data: dataInfo}); err != nil {
		h.logger.Error("error on encode response", "err", err)
	}
}

func (h *serverHandler) GetData(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dataID := id
	if dataID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "invalid data ID")
		return
	}

	result, err := h.dataService.GetData(ctx, dataID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrDataNotFound) {
			h.writeError(w, http.StatusNotFound, "data not found")
			return
		}
		if errors.Is(err, apperrors.ErrDataAccessDenied) {
			h.writeError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("error on get data", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	dataInfo, err := convertDataToDataInfo(result)
	if err != nil {
		h.logger.Error("error on convert data to info", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dataInfo)
}

func (h *serverHandler) ListData(w http.ResponseWriter, r *http.Request, params server.ListDataParams) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page := 1
	if params.Page != nil {
		page = *params.Page
	}
	pageSize := defaultPageSize
	if params.PageSize != nil {
		pageSize = *params.PageSize
	}

	var dataTypes []model.DataType
	if params.Types != nil {
		for i := range *params.Types {
			dt, err := convertDataTypeFromAPI((*params.Types)[i])
			if err != nil {
				h.writeError(w, http.StatusBadRequest, "invalid data type: "+string((*params.Types)[i]))
				return
			}
			dataTypes = append(dataTypes, dt)
		}
	}

	items, total, err := h.dataService.ListData(ctx, userID, page, pageSize, dataTypes, params.Query)
	if err != nil {
		h.logger.Error("error on list data", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	apiItems := make([]server.DataInfo, 0, len(items))
	for i := range items {
		info, err := convertDataToDataInfo(items[i])
		if err != nil {
			h.logger.Error("error on convert data to info", "err", err, "index", i)
			continue
		}
		apiItems = append(apiItems, *info)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(server.ListDataResponse{
		Items:      apiItems,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func (h *serverHandler) UpdateData(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dataID := id
	if dataID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "invalid data ID")
		return
	}

	var req server.UpdateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Data == nil {
		h.writeError(w, http.StatusBadRequest, "data is required")
		return
	}

	discriminator, err := req.Data.Discriminator()
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data type")
		return
	}
	dataType, err := convertDataTypeFromAPI(server.DataType(discriminator))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data type: "+discriminator)
		return
	}

	payloadJSON, err := req.Data.MarshalJSON()
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data payload")
		return
	}

	var metadata string
	val, valErr := req.Data.ValueByDiscriminator()
	if valErr == nil {
		switch d := val.(type) {
		case server.LoginPasswordData:
			if d.Metadata != nil {
				metadata = *d.Metadata
			}
		case server.BankCardData:
			if d.Metadata != nil {
				metadata = *d.Metadata
			}
		case server.TextData:
			if d.Metadata != nil {
				metadata = *d.Metadata
			}
		}
	}

	result, err := h.dataService.UpdateData(ctx, dataID, userID, payloadJSON, dataType, metadata)
	if err != nil {
		if errors.Is(err, apperrors.ErrDataNotFound) {
			h.writeError(w, http.StatusNotFound, "data not found")
			return
		}
		if errors.Is(err, apperrors.ErrDataAccessDenied) {
			h.writeError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("error on update data", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	dataInfo, err := convertDataToDataInfo(result)
	if err != nil {
		h.logger.Error("error on convert data to info", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(server.UpdateDataResponse{Data: dataInfo})
}

func (h *serverHandler) DeleteData(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dataID := id
	if dataID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "invalid data ID")
		return
	}

	err := h.dataService.DeleteData(ctx, dataID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrDataNotFound) {
			h.writeError(w, http.StatusNotFound, "data not found")
			return
		}
		if errors.Is(err, apperrors.ErrDataAccessDenied) {
			h.writeError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("error on delete data", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
