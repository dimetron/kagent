package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apperrors "github.com/kagent-dev/kagent/go/pkg/app/errors"
)

const (
	MaxLineLength = 2000
	MaxFileSize   = 100 * 1024 * 1024 // 100 MB
)

// ReadFileTool implements file reading with line numbers
type ReadFileTool struct {
	BaseTool
}

// NewReadFileTool creates a new ReadFileTool
func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{
		BaseTool: NewBaseTool("read_file", "Read a file with line numbers"),
	}
}

func (r *ReadFileTool) RunAsync(ctx context.Context, args map[string]interface{}, toolCtx *Context) (string, error) {
	filePath, ok := args["path"].(string)
	if !ok {
		return "", apperrors.New(apperrors.ErrCodeInvalidInput, "path is required", nil)
	}

	// Resolve path relative to session directory
	if !filepath.IsAbs(filePath) && toolCtx.SessionPath != "" {
		filePath = filepath.Join(toolCtx.SessionPath, filePath)
	}

	// Clean path to prevent directory traversal
	filePath = filepath.Clean(filePath)

	// Check file size
	info, err := os.Stat(filePath)
	if err != nil {
		return "", apperrors.New(apperrors.ErrCodeFileOperation, "failed to stat file", err)
	}
	if info.Size() > MaxFileSize {
		return "", apperrors.New(apperrors.ErrCodeFileOperation,
			fmt.Sprintf("file too large: %d bytes (max: %d)", info.Size(), MaxFileSize), nil)
	}

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return "", apperrors.New(apperrors.ErrCodeFileOperation, "failed to open file", err)
	}
	defer file.Close()

	// Get offset and limit
	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}
	limit := -1
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Read lines with line numbers
	var result strings.Builder
	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		if lineNum < offset {
			lineNum++
			continue
		}
		if limit > 0 && lineNum >= offset+limit {
			break
		}

		line := scanner.Text()
		// Truncate long lines
		if len(line) > MaxLineLength {
			line = line[:MaxLineLength] + "... (truncated)"
		}

		result.WriteString(fmt.Sprintf("%5d\t%s\n", lineNum, line))
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return "", apperrors.New(apperrors.ErrCodeFileOperation, "failed to read file", err)
	}

	return result.String(), nil
}

// WriteFileTool implements file writing
type WriteFileTool struct {
	BaseTool
}

// NewWriteFileTool creates a new WriteFileTool
func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{
		BaseTool: NewBaseTool("write_file", "Write content to a file"),
	}
}

func (w *WriteFileTool) RunAsync(ctx context.Context, args map[string]interface{}, toolCtx *Context) (string, error) {
	filePath, ok := args["path"].(string)
	if !ok {
		return "", apperrors.New(apperrors.ErrCodeInvalidInput, "path is required", nil)
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", apperrors.New(apperrors.ErrCodeInvalidInput, "content is required", nil)
	}

	// Resolve path relative to session directory
	if !filepath.IsAbs(filePath) && toolCtx.SessionPath != "" {
		filePath = filepath.Join(toolCtx.SessionPath, filePath)
	}

	// Clean path to prevent directory traversal
	filePath = filepath.Clean(filePath)

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", apperrors.New(apperrors.ErrCodeFileOperation, "failed to create directory", err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", apperrors.New(apperrors.ErrCodeFileOperation, "failed to write file", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), filePath), nil
}

// EditFileTool implements file editing with string replacement
type EditFileTool struct {
	BaseTool
}

// NewEditFileTool creates a new EditFileTool
func NewEditFileTool() *EditFileTool {
	return &EditFileTool{
		BaseTool: NewBaseTool("edit_file", "Edit a file by replacing strings"),
	}
}

func (e *EditFileTool) RunAsync(ctx context.Context, args map[string]interface{}, toolCtx *Context) (string, error) {
	filePath, ok := args["path"].(string)
	if !ok {
		return "", apperrors.New(apperrors.ErrCodeInvalidInput, "path is required", nil)
	}

	oldString, ok := args["old_string"].(string)
	if !ok {
		return "", apperrors.New(apperrors.ErrCodeInvalidInput, "old_string is required", nil)
	}

	newString, ok := args["new_string"].(string)
	if !ok {
		return "", apperrors.New(apperrors.ErrCodeInvalidInput, "new_string is required", nil)
	}

	replaceAll := false
	if ra, ok := args["replace_all"].(bool); ok {
		replaceAll = ra
	}

	// Resolve path relative to session directory
	if !filepath.IsAbs(filePath) && toolCtx.SessionPath != "" {
		filePath = filepath.Join(toolCtx.SessionPath, filePath)
	}

	// Clean path to prevent directory traversal
	filePath = filepath.Clean(filePath)

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", apperrors.New(apperrors.ErrCodeFileOperation, "failed to read file", err)
	}

	// Perform replacement
	contentStr := string(content)
	var newContent string
	var count int

	if replaceAll {
		count = strings.Count(contentStr, oldString)
		newContent = strings.ReplaceAll(contentStr, oldString, newString)
	} else {
		// Replace only first occurrence
		if !strings.Contains(contentStr, oldString) {
			return "", apperrors.New(apperrors.ErrCodeInvalidInput,
				"old_string not found in file", nil)
		}
		// Check for uniqueness
		if strings.Count(contentStr, oldString) > 1 {
			return "", apperrors.New(apperrors.ErrCodeInvalidInput,
				"old_string is not unique (use replace_all=true to replace all)", nil)
		}
		newContent = strings.Replace(contentStr, oldString, newString, 1)
		count = 1
	}

	// Write back
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return "", apperrors.New(apperrors.ErrCodeFileOperation, "failed to write file", err)
	}

	return fmt.Sprintf("Successfully replaced %d occurrence(s) in %s", count, filePath), nil
}
