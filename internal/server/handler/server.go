package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

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
	fileService FileService
	logger      intlogger.Logger
}

func NewServerHandler(
	authService AuthService,
	dataService DataService,
	fileService FileService,
	logger intlogger.Logger,
) server.ServerInterface {
	return &serverHandler{
		authService: authService,
		dataService: dataService,
		fileService: fileService,
		logger:      logger,
	}
}

func (h *serverHandler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := server.HealthResponse{
		Status: "ok",
	}

	h.jsonEncode(w, response)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := convertTokenToAPI(token)

	h.jsonEncode(w, response)
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

	w.Header().Set("Content-Type", "application/json")

	response := convertTokenToAPI(token)

	h.jsonEncode(w, response)
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

	w.Header().Set("Content-Type", "application/json")

	response := convertTokenToAPI(token)

	h.jsonEncode(w, response)
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

	dataType, err := convertDataTypeFromAPIData(req.Data)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data type")
		return
	}

	payloadJSON, err := req.Data.MarshalJSON()
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data payload")
		return
	}

	metadata := convertDataMetadataFromAPIData(req.Data)
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

	response := server.CreateDataResponse{Data: dataInfo}

	h.jsonEncode(w, response)
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

	h.jsonEncode(w, dataInfo)
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

	response := server.ListDataResponse{
		Items:      apiItems,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	h.jsonEncode(w, response)
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

	dataType, err := convertDataTypeFromAPIData(req.Data)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data type")
		return
	}

	payloadJSON, err := req.Data.MarshalJSON()
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid data payload")
		return
	}

	metadata := convertDataMetadataFromAPIData(req.Data)
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

	response := server.UpdateDataResponse{Data: dataInfo}

	h.jsonEncode(w, response)
}

// nolint: dupl
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

func (h *serverHandler) InitUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req server.FileInitRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.MimeType == "" || req.Size < 0 || req.ChunksCount < 1 {
		h.writeError(w, http.StatusBadRequest, "invalid request parameters")
		return
	}

	file, err := h.fileService.InitUpload(ctx, userID, req.Name, req.MimeType, req.Size, req.ChunksCount)
	if err != nil {
		h.logger.Error("error on init upload", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := server.FileInitResponse{
		FileId:      file.ID,
		ChunksCount: file.ChunksCount,
	}

	h.jsonEncode(w, response)
}

func (h *serverHandler) UploadChunk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	fileIDStr := r.FormValue("file_id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid file_id")
		return
	}

	chunkIndexStr := r.FormValue("chunk_index")
	chunkIndex := 0
	if chunkIndexStr != "" {
		n, errAtoi := strconv.Atoi(chunkIndexStr)
		if errAtoi != nil {
			h.writeError(w, http.StatusBadRequest, "invalid chunk_index")
			return
		}
		chunkIndex = n
	}

	dataFile, _, err := r.FormFile("data")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "missing data file")
		return
	}
	defer func() {
		errClose := dataFile.Close()
		if errClose != nil {
			h.logger.Error("error on close data file", "err", errClose)
		}
	}()

	data, err := io.ReadAll(dataFile)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to read chunk data")
		return
	}

	err = h.fileService.UploadChunk(ctx, fileID, userID, chunkIndex, data)
	if err != nil {
		if errors.Is(err, apperrors.ErrFileNotFound) {
			h.writeError(w, http.StatusNotFound, "file not found")
			return
		}
		if errors.Is(err, apperrors.ErrFileAccessDenied) {
			h.writeError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("error on upload chunk", "err", err)
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := server.FileChunkResponse{
		FileId:     fileID,
		ChunkIndex: chunkIndex,
		Uploaded:   true,
	}

	h.jsonEncode(w, response)
}

func (h *serverHandler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req server.FileCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fileID := req.FileId
	if fileID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "invalid file_id")
		return
	}

	file, err := h.fileService.CompleteUpload(ctx, fileID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrFileNotFound) {
			h.writeError(w, http.StatusNotFound, "file not found")
			return
		}
		if errors.Is(err, apperrors.ErrFileAccessDenied) {
			h.writeError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("error on complete upload", "err", err)
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")

	fileInfo, err := convertFileToAPI(file)
	if err != nil {
		h.logger.Error("error on convert file", "err", err)
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.jsonEncode(w, fileInfo)
}

func (h *serverHandler) DownloadFile(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileID := id
	if fileID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "invalid file ID")
		return
	}

	reader, mimeType, size, err := h.fileService.DownloadFile(ctx, fileID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrFileNotFound) {
			h.writeError(w, http.StatusNotFound, "file not found")
			return
		}
		if errors.Is(err, apperrors.ErrFileAccessDenied) {
			h.writeError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("error on download file", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	defer func() {
		errClose := reader.Close()
		if errClose != nil {
			h.logger.Error("error on close data file", "err", errClose)
		}
	}()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

// nolint: dupl
func (h *serverHandler) DeleteFile(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	ctx := r.Context()
	userID, ok := authctx.GetUserIDFromContext(ctx)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileID := id
	if fileID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "invalid file ID")
		return
	}

	err := h.fileService.DeleteFile(ctx, fileID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrFileNotFound) {
			h.writeError(w, http.StatusNotFound, "file not found")
			return
		}
		if errors.Is(err, apperrors.ErrFileAccessDenied) {
			h.writeError(w, http.StatusForbidden, "access denied")
			return
		}
		h.logger.Error("error on delete file", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *serverHandler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	h.jsonEncode(w, server.Error{Message: message})
}

func (h *serverHandler) jsonEncode(w http.ResponseWriter, v any) {
	enc := json.NewEncoder(w)
	err := enc.Encode(v)
	if err != nil {
		h.logger.Error("error on encode response", "err", err)
		h.writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}
