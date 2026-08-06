package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/liebeSonne/gophkeeper/internal/client/app"
	"github.com/liebeSonne/gophkeeper/internal/client/model"
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
	if a.Storage == nil {
		return fmt.Errorf("storage not configured")
	}

	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	types, _ := cmd.Flags().GetStringSlice("type")
	query, _ := cmd.Flags().GetString("search")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if pageSize > 100 {
		pageSize = 100
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * pageSize

	entries, err := a.Storage.DataList(model.DataFilter{
		Types:  types,
		Query:  query,
		Limit:  pageSize,
		Offset: offset,
	})
	if err != nil {
		return fmt.Errorf("list data: %w", err)
	}

	return outputDataList(entries, jsonOutput, page, pageSize)
}

func runDataGet(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}
	if a.Storage == nil {
		return fmt.Errorf("storage not configured")
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

	entry, err := a.Storage.DataGet(id)
	if err != nil {
		return fmt.Errorf("get data: %w", err)
	}

	return outputDataGet(entry, jsonOutput)
}

func runDataCreate(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}
	if a.Storage == nil {
		return fmt.Errorf("storage not configured")
	}

	dataType, _ := cmd.Flags().GetString("type")
	payloadStr, _ := cmd.Flags().GetString("payload")
	payloadFile, _ := cmd.Flags().GetString("payload-file")

	if dataType == "" {
		return fmt.Errorf("--type is required")
	}

	payload, err := readPayload(payloadStr, payloadFile)
	if err != nil {
		return err
	}

	now := time.Now()
	entry := model.DataEntry{
		ID:         uuid.New(),
		Type:       dataType,
		Payload:    string(payload),
		SyncStatus: model.SyncStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := a.Storage.DataPut(entry); err != nil {
		return fmt.Errorf("save data: %w", err)
	}

	fmt.Printf("Data created successfully. ID: %s\n", entry.ID)
	return nil
}

func runDataUpdate(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}
	if a.Storage == nil {
		return fmt.Errorf("storage not configured")
	}

	idStr, _ := cmd.Flags().GetString("id")
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

	payloadStr, _ := cmd.Flags().GetString("payload")
	payloadFile, _ := cmd.Flags().GetString("payload-file")

	payload, err := readPayload(payloadStr, payloadFile)
	if err != nil {
		return err
	}

	existing, err := a.Storage.DataGet(id)
	if err != nil {
		return fmt.Errorf("get data: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("data entry not found: %s", idStr)
	}

	entry := model.DataEntry{
		ID:         existing.ID,
		RemoteID:   existing.RemoteID,
		Type:       dataType,
		Payload:    string(payload),
		SyncStatus: model.SyncStatusPending,
		ServerEtag: existing.ServerEtag,
		CreatedAt:  existing.CreatedAt,
		UpdatedAt:  time.Now(),
	}

	if err := a.Storage.DataPut(entry); err != nil {
		return fmt.Errorf("save data: %w", err)
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
	if a.Storage == nil {
		return fmt.Errorf("storage not configured")
	}

	idStr, _ := cmd.Flags().GetString("id")
	if idStr == "" {
		return fmt.Errorf("--id is required")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID format: %w", err)
	}

	existing, err := a.Storage.DataGet(id)
	if err != nil {
		return fmt.Errorf("get data: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("data entry not found: %s", idStr)
	}

	if existing.SyncStatus == model.SyncStatusPending && existing.RemoteID == nil {
		if err := a.Storage.DataDelete(id); err != nil {
			return fmt.Errorf("delete data: %w", err)
		}
		fmt.Println("Data deleted successfully.")
		return nil
	}

	entry := model.DataEntry{
		ID:         existing.ID,
		RemoteID:   existing.RemoteID,
		Type:       existing.Type,
		Payload:    existing.Payload,
		SyncStatus: model.SyncStatusDeleting,
		ServerEtag: existing.ServerEtag,
		CreatedAt:  existing.CreatedAt,
		UpdatedAt:  time.Now(),
	}

	if err := a.Storage.DataPut(entry); err != nil {
		return fmt.Errorf("save data: %w", err)
	}

	fmt.Println("Data marked for deletion. Run 'gk sync' to delete from server.")
	return nil
}

func readPayload(payloadStr, payloadFile string) ([]byte, error) {
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

	return payload, nil
}

func outputDataList(entries []model.DataEntry, jsonOutput bool, page, pageSize int) error {
	if jsonOutput {
		type listOutput struct {
			Items      []model.DataEntry `json:"items"`
			Page       int               `json:"page"`
			TotalPages int               `json:"total_pages"`
			Total      int               `json:"total"`
		}
		totalPages := 1
		if pageSize > 0 {
			totalPages = (len(entries) + pageSize - 1) / pageSize
		}
		output := listOutput{
			Items:      entries,
			Page:       page,
			TotalPages: totalPages,
			Total:      len(entries),
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	if len(entries) == 0 {
		fmt.Println("No data entries found.")
		return nil
	}

	fmt.Printf("%-36s %-18s %-10s %s\n", "ID", "Type", "Sync", "Updated")
	fmt.Println("------------------------------------------------------------------------")

	for _, entry := range entries {
		fmt.Printf("%-36s %-18s %-10s %s\n",
			entry.ID.String(),
			entry.Type,
			string(entry.SyncStatus),
			entry.UpdatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	fmt.Printf("\nShowing %d entries\n", len(entries))
	return nil
}

func outputDataGet(entry *model.DataEntry, jsonOutput bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(entry)
	}

	if entry == nil {
		fmt.Println("Data entry not found.")
		return nil
	}

	fmt.Printf("ID:       %s\n", entry.ID)
	fmt.Printf("Type:     %s\n", entry.Type)
	fmt.Printf("Sync:     %s\n", entry.SyncStatus)
	fmt.Printf("Created:  %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:  %s\n", entry.UpdatedAt.Format("2006-01-02 15:04:05"))

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(entry.Payload), &payload); err == nil {
		fmt.Println("\nPayload:")
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(payload)
	}

	if entry.ErrorMessage != nil {
		fmt.Printf("\nError: %s\n", *entry.ErrorMessage)
	}

	return nil
}
