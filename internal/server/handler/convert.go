package handler

import (
	"encoding/json"
	"fmt"

	server "github.com/liebeSonne/gophkeeper/api/swagger"
	"github.com/liebeSonne/gophkeeper/internal/model"
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
	default:
		return 0, fmt.Errorf("unknown data type: %s", dt)
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
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal login_password payload: %w", err)
		}
		var metadata *string
		if data.Metadata != "" {
			metadata = strToPtr(data.Metadata)
		}
		info = &server.DataInfo{}
		_ = info.FromLoginPasswordDataInfo(server.LoginPasswordDataInfo{
			Id:        data.ID,
			Type:      server.LoginPasswordDataInfoTypeLOGINPASSWORD,
			Login:     p.Login,
			Password:  p.Password,
			Metadata:  metadata,
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		})
	case model.DataTypeBankCard:
		var p model.BankCardPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal bank_card payload: %w", err)
		}
		var metadata *string
		if data.Metadata != "" {
			metadata = strToPtr(data.Metadata)
		}
		info = &server.DataInfo{}
		_ = info.FromBankCardDataInfo(server.BankCardDataInfo{
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
	case model.DataTypeText:
		var p model.TextPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal text payload: %w", err)
		}
		var metadata *string
		if data.Metadata != "" {
			metadata = strToPtr(data.Metadata)
		}
		info = &server.DataInfo{}
		_ = info.FromTextDataInfo(server.TextDataInfo{
			Id:        data.ID,
			Type:      server.TextDataInfoTypeTEXT,
			Text:      p.Text,
			Metadata:  metadata,
			CreatedAt: data.CreatedAt,
			UpdatedAt: data.UpdatedAt,
		})
	default:
		return nil, fmt.Errorf("unknown data type: %d", data.Type)
	}

	return info, nil
}
