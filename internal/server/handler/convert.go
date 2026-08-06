package handler

import (
	"encoding/json"
	"fmt"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	"github.com/liebeSonne/gophkeeper/internal/server/model"
)

func strToPtr(s string) *string {
	return &s
}

func convertTokenToAPI(token model.Token) server.TokenResponse {
	return server.TokenResponse{
		AccessToken:           token.AccessToken,
		ExpiresIn:             token.ExpiresIn,
		TokenType:             "Bearer",
		RefreshToken:          token.RefreshToken,
		RefreshTokenExpiresIn: token.RefreshTokenExpiresIn,
	}
}

func convertDataTypeFromAPI(dt server.DataType) (model.DataType, error) {
	switch dt {
	case server.DataTypeLOGINPASSWORD:
		return model.DataTypeLoginPassword, nil
	case server.DataTypeBANKCARD:
		return model.DataTypeBankCard, nil
	case server.DataTypeTEXT:
		return model.DataTypeText, nil
	case server.DataTypeFILE:
		return model.DataTypeFile, nil
	default:
		return "", fmt.Errorf("unknown data type: %s", dt)
	}
}

func convertDataToDataInfo(data model.Data) (*server.DataInfo, error) {
	payload := data.Payload
	if len(payload) == 0 {
		return nil, fmt.Errorf("invalid payload")
	}

	var info *server.DataInfo

	switch data.Type {
	case model.DataTypeLoginPassword:
		var p model.LoginPasswordPayload
		err := json.Unmarshal(payload, &p)
		if err != nil {
			return nil, fmt.Errorf("unmarshal login_password payload: %w", err)
		}
		var metadata *string
		if data.Metadata != "" {
			metadata = strToPtr(data.Metadata)
		}
		info = &server.DataInfo{}
		err = info.FromLoginPasswordDataInfo(server.LoginPasswordDataInfo{
			Id:        data.ID,
			Type:      server.LoginPasswordDataInfoTypeLOGINPASSWORD,
			Login:     p.Login,
			Password:  p.Password,
			Metadata:  metadata,
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("error on create data info: %w", err)
		}
	case model.DataTypeBankCard:
		var p model.BankCardPayload
		err := json.Unmarshal(payload, &p)
		if err != nil {
			return nil, fmt.Errorf("unmarshal bank_card payload: %w", err)
		}
		var metadata *string
		if data.Metadata != "" {
			metadata = strToPtr(data.Metadata)
		}
		info = &server.DataInfo{}
		err = info.FromBankCardDataInfo(server.BankCardDataInfo{
			Id:         data.ID,
			Type:       server.BankCardDataInfoTypeBANKCARD,
			CardNumber: p.CardNumber,
			CardHolder: p.CardHolder,
			CardExpiry: p.CardExpiry,
			CardCvv:    p.CardCVV,
			Metadata:   metadata,
			CreatedAt:  data.CreatedAt,
			UpdatedAt:  data.UpdatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("error on create data info: %w", err)
		}
	case model.DataTypeText:
		var p model.TextPayload
		err := json.Unmarshal(payload, &p)
		if err != nil {
			return nil, fmt.Errorf("unmarshal text payload: %w", err)
		}
		var metadata *string
		if data.Metadata != "" {
			metadata = strToPtr(data.Metadata)
		}
		info = &server.DataInfo{}
		err = info.FromTextDataInfo(server.TextDataInfo{
			Id:        data.ID,
			Type:      server.TextDataInfoTypeTEXT,
			Text:      p.Text,
			Metadata:  metadata,
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("error on create data info: %w", err)
		}
	case model.DataTypeFile:
		var p model.FilePayload
		err := json.Unmarshal(payload, &p)
		if err != nil {
			return nil, fmt.Errorf("unmarshal file payload: %w", err)
		}
		var metadata *string
		if data.Metadata != "" {
			metadata = strToPtr(data.Metadata)
		}

		info = &server.DataInfo{}
		err = info.FromFileDataInfo(server.FileDataInfo{
			Id:        data.ID,
			Type:      server.FileDataInfoTypeFILE,
			FileIds:   p.FileIDs,
			Metadata:  metadata,
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("error on create data info: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown data type: %s", data.Type)
	}

	return info, nil
}

func convertFileToAPI(file model.File) (server.FileInfo, error) {
	status, err := convertFileStatusToAPI(file.Status)
	if err != nil {
		return server.FileInfo{}, fmt.Errorf("error converting file status: %w", err)
	}
	return server.FileInfo{
		Id:          file.ID,
		Name:        file.Name,
		MimeType:    file.MimeType,
		Size:        file.Size,
		ChunksCount: file.ChunksCount,
		Status:      status,
		CreatedAt:   file.CreatedAt,
		UpdatedAt:   file.UpdatedAt,
	}, nil
}

func convertFileStatusToAPI(status model.FileStatus) (server.FileStatus, error) {
	switch status {
	case model.FileStatusCompleted:
		return server.COMPLETED, nil
	case model.FileStatusFailed:
		return server.FAILED, nil
	case model.FileStatusInProgress:
		return server.INPROGRESS, nil
	default:
		return "", fmt.Errorf("unknown file stataus: %s", status)
	}
}

func convertDataMetadataFromAPIData(data *server.Data) string {
	var metadata string
	val, valErr := data.ValueByDiscriminator()
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
		case server.FileData:
			if d.Metadata != nil {
				metadata = *d.Metadata
			}
		}
	}
	return metadata
}

func convertDataTypeFromAPIData(data *server.Data) (model.DataType, error) {
	discriminator, err := data.Discriminator()
	if err != nil {
		return "", fmt.Errorf("invalid data type: %w", err)
	}
	dataType, err := convertDataTypeFromAPI(server.DataType(discriminator))

	if err != nil {
		return "", fmt.Errorf("invalid data type %s: %w", discriminator, err)
	}
	return dataType, nil
}
