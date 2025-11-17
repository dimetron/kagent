package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kagent-dev/kagent/go/pkg/adk/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFileTool_Name(t *testing.T) {
	tool := NewReadFileTool()
	assert.Equal(t, "READ_FILE", tool.Name())
}

func TestReadFileTool_Description(t *testing.T) {
	tool := NewReadFileTool()
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "read")
}

func TestReadFileTool_RunAsync_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nLine 2\nLine 3"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Create tool context
	toolCtx := &Context{
		Session: &session.Session{
			ID:      "test-session",
			UserID:  "test-user",
			AppName: "test-app",
		},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	// Run tool
	tool := NewReadFileTool()
	result, err := tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path": testFile,
	}, toolCtx)

	require.NoError(t, err)
	assert.Contains(t, result, "Hello, World!")
	assert.Contains(t, result, "Line 2")
	assert.Contains(t, result, "Line 3")
}

func TestReadFileTool_RunAsync_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewReadFileTool()
	_, err := tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path": "/nonexistent/file.txt",
	}, toolCtx)

	require.Error(t, err)
}

func TestReadFileTool_RunAsync_MissingFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewReadFileTool()
	_, err := tool.RunAsync(context.Background(), map[string]interface{}{}, toolCtx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "file_path")
}

func TestWriteFileTool_Name(t *testing.T) {
	tool := NewWriteFileTool()
	assert.Equal(t, "WRITE_FILE", tool.Name())
}

func TestWriteFileTool_Description(t *testing.T) {
	tool := NewWriteFileTool()
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "write")
}

func TestWriteFileTool_RunAsync_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "output.txt")

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewWriteFileTool()
	testContent := "This is test content"

	result, err := tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path": testFile,
		"content":   testContent,
	}, toolCtx)

	require.NoError(t, err)
	assert.Contains(t, result, "successfully")

	// Verify file was created with correct content
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestWriteFileTool_RunAsync_MissingContent(t *testing.T) {
	tmpDir := t.TempDir()

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewWriteFileTool()
	_, err := tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path": "/tmp/test.txt",
	}, toolCtx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "content")
}

func TestWriteFileTool_RunAsync_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "nested", "output.txt")

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewWriteFileTool()
	result, err := tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path": testFile,
		"content":   "nested content",
	}, toolCtx)

	require.NoError(t, err)
	assert.Contains(t, result, "successfully")

	// Verify directories were created
	assert.FileExists(t, testFile)
}

func TestEditFileTool_Name(t *testing.T) {
	tool := NewEditFileTool()
	assert.Equal(t, "EDIT_FILE", tool.Name())
}

func TestEditFileTool_Description(t *testing.T) {
	tool := NewEditFileTool()
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "edit")
}

func TestEditFileTool_RunAsync_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "edit.txt")
	originalContent := "Hello World\nThis is a test\nGoodbye World"
	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	require.NoError(t, err)

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewEditFileTool()
	result, err := tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path":  testFile,
		"old_string": "This is a test",
		"new_string": "This is edited",
	}, toolCtx)

	require.NoError(t, err)
	assert.Contains(t, result, "successfully")

	// Verify content was edited
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "This is edited")
	assert.NotContains(t, string(content), "This is a test")
}

func TestEditFileTool_RunAsync_OldStringNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "edit.txt")
	err := os.WriteFile(testFile, []byte("Hello World"), 0644)
	require.NoError(t, err)

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewEditFileTool()
	_, err = tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path":  testFile,
		"old_string": "nonexistent text",
		"new_string": "replacement",
	}, toolCtx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEditFileTool_RunAsync_MissingParameters(t *testing.T) {
	tmpDir := t.TempDir()

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewEditFileTool()

	// Missing old_string
	_, err := tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path":  "/tmp/test.txt",
		"new_string": "replacement",
	}, toolCtx)
	require.Error(t, err)

	// Missing new_string
	_, err = tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path":  "/tmp/test.txt",
		"old_string": "old",
	}, toolCtx)
	require.Error(t, err)
}

func TestReadFileTool_LongFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with many lines
	testFile := filepath.Join(tmpDir, "long.txt")
	f, err := os.Create(testFile)
	require.NoError(t, err)
	defer f.Close()

	for i := 0; i < 5000; i++ {
		f.WriteString("Line " + string(rune(i)) + "\n")
	}

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewReadFileTool()
	result, err := tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path": testFile,
	}, toolCtx)

	require.NoError(t, err)
	// Should include truncation message or limit lines
	assert.NotEmpty(t, result)
}

func TestWriteFileTool_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "overwrite.txt")

	// Write initial content
	err := os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	toolCtx := &Context{
		Session:     &session.Session{ID: "test"},
		SessionPath: tmpDir,
		InvocationContext: &session.InvocationContext{
			SessionID: "test-session",
		},
	}

	tool := NewWriteFileTool()
	_, err = tool.RunAsync(context.Background(), map[string]interface{}{
		"file_path": testFile,
		"content":   "new content",
	}, toolCtx)

	require.NoError(t, err)

	// Verify content was overwritten
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))
}
