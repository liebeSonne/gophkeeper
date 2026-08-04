package gophkeeper

import (
	"context"
	"errors"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

type RefreshFunc func(ctx context.Context) error

type Client struct {
	*gophkeeper.ClientWithResponses
	authToken   string
	refreshFunc RefreshFunc
}

func NewClient(baseURL string) (*Client, error) {
	client, err := gophkeeper.NewClientWithResponses(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		ClientWithResponses: client,
	}, nil
}

func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}

func (c *Client) SetRefreshToken(refreshFn RefreshFunc) {
	c.refreshFunc = refreshFn
}

func (c *Client) authEditor(_ context.Context, req *http.Request) error {
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	return nil
}

func (c *Client) HealthCheck(ctx context.Context) (*gophkeeper.HealthCheckResponse, error) {
	resp, err := c.HealthCheckWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) RegisterUser(ctx context.Context, login, password string) (*gophkeeper.RegisterUserResponse, error) {
	resp, err := c.RegisterUserWithResponse(ctx, gophkeeper.RegisterUserJSONRequestBody{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) LoginUser(ctx context.Context, login, password string) (*gophkeeper.LoginUserResponse, error) {
	resp, err := c.LoginUserWithResponse(ctx, gophkeeper.LoginUserJSONRequestBody{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) RefreshTokenAPI(ctx context.Context, refreshToken string) (*gophkeeper.RefreshTokenResponse, error) {
	resp, err := c.RefreshTokenWithResponse(ctx, gophkeeper.RefreshTokenJSONRequestBody{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) CreateData(ctx context.Context, data gophkeeper.Data) (*gophkeeper.CreateDataResponse, error) {
	resp, err := c.CreateDataWithResponse(ctx, gophkeeper.CreateDataJSONRequestBody{
		Data: &data,
	}, c.authEditor)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return nil, refreshErr
		}
		resp, err = c.CreateDataWithResponse(ctx, gophkeeper.CreateDataJSONRequestBody{
			Data: &data,
		}, c.authEditor)
		if err != nil {
			return nil, err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) GetData(ctx context.Context, id openapi_types.UUID) (*gophkeeper.GetDataResponse, error) {
	resp, err := c.GetDataWithResponse(ctx, id, c.authEditor)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return nil, refreshErr
		}
		resp, err = c.GetDataWithResponse(ctx, id, c.authEditor)
		if err != nil {
			return nil, err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) ListData(ctx context.Context, page, pageSize *int, types *[]gophkeeper.DataType, query *string) (*gophkeeper.ListDataResponse, error) {
	params := buildListDataParams(page, pageSize, types, query)

	resp, err := c.ListDataWithResponse(ctx, params, c.authEditor)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return nil, refreshErr
		}
		resp, err = c.ListDataWithResponse(ctx, params, c.authEditor)
		if err != nil {
			return nil, err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) UpdateData(ctx context.Context, id openapi_types.UUID, data gophkeeper.Data) (*gophkeeper.UpdateDataResponse, error) {
	resp, err := c.UpdateDataWithResponse(ctx, id, gophkeeper.UpdateDataJSONRequestBody{
		Data: &data,
	}, c.authEditor)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return nil, refreshErr
		}
		resp, err = c.UpdateDataWithResponse(ctx, id, gophkeeper.UpdateDataJSONRequestBody{
			Data: &data,
		}, c.authEditor)
		if err != nil {
			return nil, err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) DeleteData(ctx context.Context, id openapi_types.UUID) error {
	resp, err := c.DeleteDataWithResponse(ctx, id, c.authEditor)
	if err != nil {
		return err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return refreshErr
		}
		resp, err = c.DeleteDataWithResponse(ctx, id, c.authEditor)
		if err != nil {
			return err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return parseError(resp)
	}

	return nil
}

func (c *Client) InitUpload(ctx context.Context, req gophkeeper.FileInitRequest) (*gophkeeper.InitUploadResponse, error) {
	resp, err := c.InitUploadWithResponse(ctx, req, c.authEditor)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return nil, refreshErr
		}
		resp, err = c.InitUploadWithResponse(ctx, req, c.authEditor)
		if err != nil {
			return nil, err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) CompleteUpload(ctx context.Context, fileID openapi_types.UUID) (*gophkeeper.CompleteUploadResponse, error) {
	resp, err := c.CompleteUploadWithResponse(ctx, gophkeeper.FileCompleteRequest{
		FileId: fileID,
	}, c.authEditor)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return nil, refreshErr
		}
		resp, err = c.CompleteUploadWithResponse(ctx, gophkeeper.FileCompleteRequest{
			FileId: fileID,
		}, c.authEditor)
		if err != nil {
			return nil, err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) DownloadFile(ctx context.Context, id openapi_types.UUID) (*gophkeeper.DownloadFileResponse, error) {
	resp, err := c.DownloadFileWithResponse(ctx, id, c.authEditor)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return nil, refreshErr
		}
		resp, err = c.DownloadFileWithResponse(ctx, id, c.authEditor)
		if err != nil {
			return nil, err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func (c *Client) DeleteFile(ctx context.Context, id openapi_types.UUID) error {
	resp, err := c.DeleteFileWithResponse(ctx, id, c.authEditor)
	if err != nil {
		return err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return refreshErr
		}
		resp, err = c.DeleteFileWithResponse(ctx, id, c.authEditor)
		if err != nil {
			return err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return parseError(resp)
	}

	return nil
}

func (c *Client) ListFiles(ctx context.Context, page, pageSize *int, query *string) (*gophkeeper.ListFilesResponse, error) {
	params := buildListFilesParams(page, pageSize, query)

	resp, err := c.ListFilesWithResponse(ctx, params, c.authEditor)
	if err != nil {
		return nil, err
	}

	if resp.HTTPResponse.StatusCode == http.StatusUnauthorized && c.refreshFunc != nil {
		if refreshErr := c.refreshFunc(ctx); refreshErr != nil {
			return nil, refreshErr
		}
		resp, err = c.ListFilesWithResponse(ctx, params, c.authEditor)
		if err != nil {
			return nil, err
		}
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		return nil, parseError(resp)
	}

	return resp, nil
}

func buildListDataParams(page, pageSize *int, types *[]gophkeeper.DataType, query *string) *gophkeeper.ListDataParams {
	params := &gophkeeper.ListDataParams{}
	if page != nil {
		params.Page = page
	}
	if pageSize != nil {
		params.PageSize = pageSize
	}
	if types != nil {
		params.Types = types
	}
	if query != nil {
		params.Query = query
	}
	return params
}

func buildListFilesParams(page, pageSize *int, query *string) *gophkeeper.ListFilesParams {
	params := &gophkeeper.ListFilesParams{}
	if page != nil {
		params.Page = page
	}
	if pageSize != nil {
		params.PageSize = pageSize
	}
	if query != nil {
		params.Query = query
	}
	return params
}

type APIError struct {
	Message string
}

func (e APIError) Error() string {
	return e.Message
}

func parseError(resp any) error { //nolint:gocognit,funlen // generated response types require many cases
	switch r := resp.(type) {
	case *gophkeeper.HealthCheckResponse:
		if r.JSON500 != nil {
			return APIError{Message: r.JSON500.Message}
		}
	case *gophkeeper.RegisterUserResponse:
		if r.JSON400 != nil {
			return APIError{Message: r.JSON400.Message}
		}
		if r.JSON409 != nil {
			return APIError{Message: r.JSON409.Message}
		}
	case *gophkeeper.LoginUserResponse:
		if r.JSON400 != nil {
			return APIError{Message: r.JSON400.Message}
		}
		if r.JSON401 != nil {
			return APIError{Message: r.JSON401.Message}
		}
	case *gophkeeper.RefreshTokenResponse:
		if r.JSON400 != nil {
			return APIError{Message: r.JSON400.Message}
		}
		if r.JSON401 != nil {
			return APIError{Message: r.JSON401.Message}
		}
	case *gophkeeper.CreateDataResponse:
		if r.JSON400 != nil {
			return APIError{Message: r.JSON400.Message}
		}
	case *gophkeeper.GetDataResponse:
		if r.JSON404 != nil {
			return APIError{Message: r.JSON404.Message}
		}
	case *gophkeeper.UpdateDataResponse:
		if r.JSON400 != nil {
			return APIError{Message: r.JSON400.Message}
		}
		if r.JSON404 != nil {
			return APIError{Message: r.JSON404.Message}
		}
	case *gophkeeper.DeleteDataResponse:
		if r.JSON404 != nil {
			return APIError{Message: r.JSON404.Message}
		}
	case *gophkeeper.InitUploadResponse:
		if r.JSON400 != nil {
			return APIError{Message: r.JSON400.Message}
		}
	case *gophkeeper.CompleteUploadResponse:
		if r.JSON400 != nil {
			return APIError{Message: r.JSON400.Message}
		}
		if r.JSON404 != nil {
			return APIError{Message: r.JSON404.Message}
		}
	case *gophkeeper.DownloadFileResponse:
		if r.JSON404 != nil {
			return APIError{Message: r.JSON404.Message}
		}
	case *gophkeeper.DeleteFileResponse:
		if r.JSON404 != nil {
			return APIError{Message: r.JSON404.Message}
		}
	}

	return errors.New("API error")
}
