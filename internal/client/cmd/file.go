package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/liebeSonne/gophkeeper/internal/client/app"
	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "File management commands",
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file",
	RunE:  runFileUpload,
}

var fileDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a file",
	RunE:  runFileDownload,
}

var fileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files",
	RunE:  runFileList,
}

var fileDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a file",
	RunE:  runFileDelete,
}

func initFileFlags() {
	fileUploadCmd.Flags().StringP("file", "f", "", "path to file to upload (required)")

	fileDownloadCmd.Flags().StringP("id", "i", "", "file ID (required)")
	fileDownloadCmd.Flags().StringP("output", "o", "", "output file path (default: print to stdout)")

	fileListCmd.Flags().IntP("page", "p", 1, "page number")
	fileListCmd.Flags().IntP("page-size", "s", 20, "page size (max 100)")
	fileListCmd.Flags().StringP("search", "q", "", "search query")
	fileListCmd.Flags().BoolP("json", "j", false, "output as JSON")

	fileDeleteCmd.Flags().StringP("id", "i", "", "file ID (required)")
}

func runFileUpload(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	a := app.Get()
	if a == nil {
		return fmt.Errorf("client not initialized: run 'gk init' first")
	}

	client, err := newAPIClient(a)
	if err != nil {
		return err
	}

	filePath, _ := cmd.Flags().GetString("file")
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	fileSize := fileInfo.Size()
	chunkSize := int64(5 * 1024 * 1024)
	if fileSize < chunkSize {
		chunkSize = fileSize
	}

	chunksCount := int(fileSize/chunkSize) + 1
	if fileSize == 0 {
		chunksCount = 1
	}

	initResp, err := client.InitUpload(cmd.Context(), gophkeeper.FileInitRequest{
		Name:        filepath.Base(filePath),
		Size:        fileSize,
		ChunksCount: chunksCount,
		MimeType:    "application/octet-stream",
	})
	if err != nil {
		return err
	}

	fileID := initResp.JSON201.FileId

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	chunk := make([]byte, chunkSize)
	chunkIndex := 0

	for {
		n, readErr := file.Read(chunk)
		if n == 0 {
			break
		}

		if uploadErr := uploadChunk(cmd.Context(), client.ClientWithResponses, fileID, chunkIndex, chunk[:n]); uploadErr != nil {
			return fmt.Errorf("upload chunk %d: %w", chunkIndex, uploadErr)
		}

		chunkIndex++
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read chunk: %w", readErr)
		}
	}

	_, err = client.CompleteUpload(cmd.Context(), fileID)
	if err != nil {
		return err
	}

	fmt.Printf("File uploaded successfully. ID: %s\n", fileID)
	return nil
}

func uploadChunk(ctx context.Context, client *gophkeeper.ClientWithResponses, fileID uuid.UUID, chunkIndex int, data []byte) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if writeErr := writer.WriteField("file_id", fileID.String()); writeErr != nil {
		return fmt.Errorf("write file_id: %w", writeErr)
	}

	if writeErr := writer.WriteField("chunk_index", fmt.Sprintf("%d", chunkIndex)); writeErr != nil {
		return fmt.Errorf("write chunk_index: %w", writeErr)
	}

	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="data"; filename="chunk%d"`, chunkIndex))
	h.Set("Content-Type", "application/octet-stream")

	part, partErr := writer.CreatePart(h)
	if partErr != nil {
		return fmt.Errorf("create data part: %w", partErr)
	}
	if _, writeErr := part.Write(data); writeErr != nil {
		return fmt.Errorf("write data: %w", writeErr)
	}

	if closeErr := writer.Close(); closeErr != nil {
		return fmt.Errorf("close multipart: %w", closeErr)
	}

	resp, respErr := client.UploadChunkWithBody(ctx, writer.FormDataContentType(), body)
	if respErr != nil {
		return respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload chunk: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func runFileDownload(cmd *cobra.Command, _ []string) error {
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

	outputPath, _ := cmd.Flags().GetString("output")

	resp, err := client.DownloadFile(cmd.Context(), id)
	if err != nil {
		return err
	}

	if outputPath != "" {
		if writeErr := os.WriteFile(outputPath, resp.Body, 0o600); writeErr != nil {
			return fmt.Errorf("write file: %w", writeErr)
		}
		fmt.Printf("File downloaded to: %s\n", outputPath)
	} else {
		os.Stdout.Write(resp.Body)
	}

	return nil
}

func runFileList(cmd *cobra.Command, _ []string) error {
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
	query, _ := cmd.Flags().GetString("search")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if pageSize > 100 {
		pageSize = 100
	}
	if page < 1 {
		page = 1
	}

	var queryPtr *string
	if query != "" {
		queryPtr = &query
	}

	resp, err := client.ListFiles(cmd.Context(), &page, &pageSize, queryPtr)
	if err != nil {
		return err
	}

	return outputFileList(resp.JSON200, jsonOutput)
}

func runFileDelete(cmd *cobra.Command, _ []string) error {
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

	if err := client.DeleteFile(cmd.Context(), id); err != nil {
		return err
	}

	fmt.Println("File deleted successfully.")
	return nil
}

func outputFileList(resp *gophkeeper.FilesList, jsonOutput bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(resp)
	}

	if resp == nil || len(resp.Items) == 0 {
		fmt.Println("No files found.")
		return nil
	}

	fmt.Printf("%-36s %-30s %-12s %s\n", "ID", "Name", "Size", "Status")
	fmt.Println("------------------------------------------------------------------------")

	for _, item := range resp.Items {
		size := safeUint64(item.Size)
		name := item.Name
		if len(name) > 30 {
			name = name[:30]
		}
		fmt.Printf("%-36s %-30s %-12s %s\n", item.Id.String(), name, humanFileSize(size), item.Status)
	}

	fmt.Printf("\nPage %d/%d (total: %d)\n", resp.Page, resp.TotalPages, resp.Total)
	return nil
}

func humanFileSize(size uint64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := uint64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func safeUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v) // nolint:gosec
}
