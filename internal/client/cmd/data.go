package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/liebeSonne/gophkeeper/internal/client/app"
	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

const (
	dataTypeLoginPassword = "LOGIN_PASSWORD"
	dataTypeBankCard      = "BANK_CARD"
	dataTypeText          = "TEXT"
	dataTypeFile          = "FILE"
)

var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "Data management commands",
}

var dataListCmd = &cobra.Command{
	Use:   "list",
	Short: "List data entries",
	RunE:  runDataList,
}

var dataGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a data entry by ID",
	RunE:  runDataGet,
}

var dataCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new data entry",
	RunE:  runDataCreate,
}

var dataUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a data entry",
	RunE:  runDataUpdate,
}

var dataDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a data entry",
	RunE:  runDataDelete,
}

func initDataFlags() {
	dataListCmd.Flags().IntP("page", "p", 1, "page number")
	dataListCmd.Flags().IntP("page-size", "s", 20, "page size (max 100)")
	dataListCmd.Flags().StringSliceP("type", "t", nil, "filter by data type (LOGIN_PASSWORD, BANK_CARD, TEXT, FILE)")
	dataListCmd.Flags().StringP("search", "q", "", "search query")
	dataListCmd.Flags().BoolP("json", "j", false, "output as JSON")

	dataGetCmd.Flags().StringP("id", "i", "", "data entry ID (required)")
	dataGetCmd.Flags().BoolP("json", "j", false, "output as JSON")

	dataCreateCmd.Flags().StringP("type", "t", "", "data type: LOGIN_PASSWORD, BANK_CARD, TEXT, FILE (required)")
	dataCreateCmd.Flags().StringP("payload", "d", "", "payload as JSON string")
	dataCreateCmd.Flags().StringP("payload-file", "f", "", "payload from JSON file")

	dataUpdateCmd.Flags().StringP("id", "i", "", "data entry ID (required)")
	dataUpdateCmd.Flags().StringP("type", "t", "", "data type: LOGIN_PASSWORD, BANK_CARD, TEXT, FILE (required)")
	dataUpdateCmd.Flags().StringP("payload", "d", "", "payload as JSON string")
	dataUpdateCmd.Flags().StringP("payload-file", "f", "", "payload from JSON file")

	dataDeleteCmd.Flags().StringP("id", "i", "", "data entry ID (required)")
}

func runDataList(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}

	client, err := newAPIClient(a)
	if err != nil {
		return err
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	typesStr, _ := cmd.Flags().GetStringSlice("type")
	query, _ := cmd.Flags().GetString("search")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if pageSize > 100 {
		pageSize = 100
	}
	if page < 1 {
		page = 1
	}

	var types []gophkeeper.DataType
	for _, t := range typesStr {
		types = append(types, gophkeeper.DataType(t))
	}

	var typesPtr *[]gophkeeper.DataType
	if len(types) > 0 {
		typesPtr = &types
	}

	var queryPtr *string
	if query != "" {
		queryPtr = &query
	}

	resp, err := client.ListData(cmd.Context(), &page, &pageSize, typesPtr, queryPtr)
	if err != nil {
		return err
	}

	return outputDataList(resp.JSON200, jsonOutput)
}

func runDataGet(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}

	client, err := newAPIClient(a)
	if err != nil {
		return err
	}

	idStr, _ := cmd.Flags().GetString("id")
	if idStr == "" {
		return fmt.Errorf("--id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")

	resp, err := client.GetData(cmd.Context(), id)
	if err != nil {
		return err
	}

	return outputDataGet(resp.JSON200, jsonOutput)
}

func runDataCreate(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}

	client, err := newAPIClient(a)
	if err != nil {
		return err
	}

	dataType, _ := cmd.Flags().GetString("type")
	payloadStr, _ := cmd.Flags().GetString("payload")
	payloadFile, _ := cmd.Flags().GetString("payload-file")

	if dataType == "" {
		return fmt.Errorf("--type is required")
	}

	data, err := buildDataPayload(dataType, payloadStr, payloadFile)
	if err != nil {
		return err
	}

	resp, err := client.CreateData(cmd.Context(), *data)
	if err != nil {
		return err
	}

	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		id, _ := extractDataID(resp.JSON201.Data)
		fmt.Printf("Data created successfully. ID: %s\n", id)
	} else {
		fmt.Println("Data created successfully.")
	}
	return nil
}

func runDataUpdate(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}

	client, err := newAPIClient(a)
	if err != nil {
		return err
	}

	idStr, _ := cmd.Flags().GetString("id")
	payloadStr, _ := cmd.Flags().GetString("payload")
	payloadFile, _ := cmd.Flags().GetString("payload-file")

	if idStr == "" {
		return fmt.Errorf("--id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	dataType, _ := cmd.Flags().GetString("type")
	if dataType == "" {
		return fmt.Errorf("--type is required")
	}

	data, err := buildDataPayload(dataType, payloadStr, payloadFile)
	if err != nil {
		return err
	}

	_, err = client.UpdateData(cmd.Context(), id, *data)
	if err != nil {
		return err
	}

	fmt.Println("Data updated successfully.")
	return nil
}

func runDataDelete(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}

	client, err := newAPIClient(a)
	if err != nil {
		return err
	}

	idStr, _ := cmd.Flags().GetString("id")
	if idStr == "" {
		return fmt.Errorf("--id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	if err := client.DeleteData(cmd.Context(), id); err != nil {
		return err
	}

	fmt.Println("Data deleted successfully.")
	return nil
}

func buildDataPayload(dataType, payloadStr, payloadFile string) (*gophkeeper.Data, error) {
	var payload []byte
	var err error

	switch {
	case payloadFile != "":
		payload, err = os.ReadFile(payloadFile)
		if err != nil {
			return nil, fmt.Errorf("read payload file: %w", err)
		}
	case payloadStr != "":
		payload = []byte(payloadStr)
	default:
		return nil, fmt.Errorf("--payload or --payload-file is required")
	}

	if !json.Valid(payload) {
		return nil, fmt.Errorf("invalid JSON payload")
	}

	data := &gophkeeper.Data{}

	switch dataType {
	case dataTypeLoginPassword:
		var lp gophkeeper.LoginPasswordData
		if err := json.Unmarshal(payload, &lp); err != nil {
			return nil, fmt.Errorf("parse payload: %w", err)
		}
		lp.Type = gophkeeper.LoginPasswordDataType(dataType)
		if err := data.FromLoginPasswordData(lp); err != nil {
			return nil, fmt.Errorf("build data: %w", err)
		}
	case dataTypeBankCard:
		var bc gophkeeper.BankCardData
		if err := json.Unmarshal(payload, &bc); err != nil {
			return nil, fmt.Errorf("parse payload: %w", err)
		}
		bc.Type = gophkeeper.BankCardDataType(dataType)
		if err := data.FromBankCardData(bc); err != nil {
			return nil, fmt.Errorf("build data: %w", err)
		}
	case dataTypeText:
		var td gophkeeper.TextData
		if err := json.Unmarshal(payload, &td); err != nil {
			return nil, fmt.Errorf("parse payload: %w", err)
		}
		td.Type = gophkeeper.TextDataType(dataType)
		if err := data.FromTextData(td); err != nil {
			return nil, fmt.Errorf("build data: %w", err)
		}
	case dataTypeFile:
		var fd gophkeeper.FileData
		if err := json.Unmarshal(payload, &fd); err != nil {
			return nil, fmt.Errorf("parse payload: %w", err)
		}
		fd.Type = gophkeeper.FileDataType(dataType)
		if err := data.FromFileData(fd); err != nil {
			return nil, fmt.Errorf("build data: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported data type: %s (use LOGIN_PASSWORD, BANK_CARD, TEXT, FILE)", dataType)
	}

	return data, nil
}

func outputDataList(resp *gophkeeper.DataList, jsonOutput bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(resp)
	}

	if resp == nil || len(resp.Items) == 0 {
		fmt.Println("No data entries found.")
		return nil
	}

	fmt.Printf("%-36s %-18s %s\n", "ID", "Type", "Created")
	fmt.Println("------------------------------------------------------------")

	for _, item := range resp.Items {
		id, dataType, createdAt := extractDataInfo(item)
		fmt.Printf("%-36s %-18s %s\n", id, dataType, createdAt.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\nPage %d/%d (total: %d)\n", resp.Page, resp.TotalPages, resp.Total)
	return nil
}

func outputDataGet(resp *gophkeeper.DataInfo, jsonOutput bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(resp)
	}

	if resp == nil {
		fmt.Println("Data entry not found.")
		return nil
	}

	id, dataType, createdAt := extractDataInfo(*resp)
	fmt.Printf("ID:     %s\n", id)
	fmt.Printf("Type:   %s\n", dataType)
	fmt.Printf("Created: %s\n", createdAt.Format("2006-01-02 15:04:05"))

	switch dataType {
	case dataTypeLoginPassword:
		if data, err := resp.AsLoginPasswordDataInfo(); err == nil {
			fmt.Printf("Login:    %s\n", data.Login)
			fmt.Printf("Password: %s\n", data.Password)
		}
	case dataTypeBankCard:
		if data, err := resp.AsBankCardDataInfo(); err == nil {
			fmt.Printf("Card:     %s\n", data.CardNumber)
			fmt.Printf("Holder:   %s\n", data.CardHolder)
			fmt.Printf("Expiry:   %s\n", data.CardExpiry)
		}
	case dataTypeText:
		if data, err := resp.AsTextDataInfo(); err == nil {
			fmt.Printf("Text:     %s\n", data.Text)
		}
	case dataTypeFile:
		if data, err := resp.AsFileDataInfo(); err == nil {
			fmt.Printf("Files:    %d\n", len(data.FileIds))
		}
	}

	return nil
}

func extractDataInfo(item gophkeeper.DataInfo) (id, dataType string, createdAt time.Time) {
	discriminator, err := item.Discriminator()
	if err != nil {
		return "", "UNKNOWN", time.Time{}
	}

	switch discriminator {
	case dataTypeLoginPassword:
		if data, err := item.AsLoginPasswordDataInfo(); err == nil {
			return data.Id.String(), dataTypeLoginPassword, data.CreatedAt
		}
	case dataTypeBankCard:
		if data, err := item.AsBankCardDataInfo(); err == nil {
			return data.Id.String(), dataTypeBankCard, data.CreatedAt
		}
	case dataTypeText:
		if data, err := item.AsTextDataInfo(); err == nil {
			return data.Id.String(), dataTypeText, data.CreatedAt
		}
	case dataTypeFile:
		if data, err := item.AsFileDataInfo(); err == nil {
			return data.Id.String(), dataTypeFile, data.CreatedAt
		}
	}

	return "", "UNKNOWN", time.Time{}
}

func extractDataID(item *gophkeeper.DataInfo) (string, error) {
	if item == nil {
		return "", fmt.Errorf("no data")
	}

	discriminator, err := item.Discriminator()
	if err != nil {
		return "", err
	}

	switch discriminator {
	case dataTypeLoginPassword:
		if data, err := item.AsLoginPasswordDataInfo(); err == nil {
			return data.Id.String(), nil
		}
	case dataTypeBankCard:
		if data, err := item.AsBankCardDataInfo(); err == nil {
			return data.Id.String(), nil
		}
	case dataTypeText:
		if data, err := item.AsTextDataInfo(); err == nil {
			return data.Id.String(), nil
		}
	case dataTypeFile:
		if data, err := item.AsFileDataInfo(); err == nil {
			return data.Id.String(), nil
		}
	}

	return "", fmt.Errorf("unknown data type: %s", discriminator)
}
