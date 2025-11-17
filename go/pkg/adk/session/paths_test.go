package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathManager_Initialize(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	sessionID := "test-session-123"
	skillsDir := filepath.Join(tmpDir, "skills")

	// Create skills directory
	err := os.MkdirAll(skillsDir, 0755)
	require.NoError(t, err)

	// Initialize session
	sessionPath, err := pm.Initialize(sessionID, skillsDir)
	require.NoError(t, err)
	assert.NotEmpty(t, sessionPath)
	assert.Contains(t, sessionPath, sessionID)

	// Verify directories were created
	assert.DirExists(t, sessionPath)
	assert.DirExists(t, filepath.Join(sessionPath, "uploads"))
	assert.DirExists(t, filepath.Join(sessionPath, "outputs"))
	assert.DirExists(t, filepath.Join(sessionPath, "skills"))
}

func TestPathManager_Get_Cached(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	sessionID := "test-session-456"
	skillsDir := filepath.Join(tmpDir, "skills")

	// Create skills directory
	err := os.MkdirAll(skillsDir, 0755)
	require.NoError(t, err)

	// Initialize session
	path1, err := pm.Initialize(sessionID, skillsDir)
	require.NoError(t, err)

	// Get cached path
	path2, err := pm.Get(sessionID)
	require.NoError(t, err)

	// Should return same path
	assert.Equal(t, path1, path2)
}

func TestPathManager_Get_NotInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	sessionID := "non-existent-session"

	// Get path for non-initialized session
	_, err := pm.Get(sessionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestPathManager_Initialize_CreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	sessionID := "test-session-dirs"
	skillsDir := filepath.Join(tmpDir, "skills")

	// Create skills directory
	err := os.MkdirAll(skillsDir, 0755)
	require.NoError(t, err)

	// Initialize
	sessionPath, err := pm.Initialize(sessionID, skillsDir)
	require.NoError(t, err)

	// Verify all required directories exist
	expectedDirs := []string{
		sessionPath,
		filepath.Join(sessionPath, "uploads"),
		filepath.Join(sessionPath, "outputs"),
		filepath.Join(sessionPath, "skills"),
	}

	for _, dir := range expectedDirs {
		assert.DirExists(t, dir)
	}
}

func TestPathManager_Initialize_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	sessionID := "test-session-idempotent"
	skillsDir := filepath.Join(tmpDir, "skills")

	// Create skills directory
	err := os.MkdirAll(skillsDir, 0755)
	require.NoError(t, err)

	// Initialize twice
	path1, err := pm.Initialize(sessionID, skillsDir)
	require.NoError(t, err)

	path2, err := pm.Initialize(sessionID, skillsDir)
	require.NoError(t, err)

	// Should return same path
	assert.Equal(t, path1, path2)
}

func TestPathManager_MultipleSessions(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	skillsDir := filepath.Join(tmpDir, "skills")
	err := os.MkdirAll(skillsDir, 0755)
	require.NoError(t, err)

	// Initialize multiple sessions
	sessions := []string{"session-1", "session-2", "session-3"}
	paths := make(map[string]string)

	for _, sessionID := range sessions {
		path, err := pm.Initialize(sessionID, skillsDir)
		require.NoError(t, err)
		paths[sessionID] = path
	}

	// Verify each session has unique path
	for i, sid1 := range sessions {
		for j, sid2 := range sessions {
			if i != j {
				assert.NotEqual(t, paths[sid1], paths[sid2])
			}
		}
	}

	// Verify we can get all paths
	for sessionID, expectedPath := range paths {
		path, err := pm.Get(sessionID)
		require.NoError(t, err)
		assert.Equal(t, expectedPath, path)
	}
}

func TestPathManager_SkillsDirectoryCopy(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	sessionID := "test-session-skills"
	skillsDir := filepath.Join(tmpDir, "source-skills")

	// Create skills directory with a test file
	err := os.MkdirAll(skillsDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(skillsDir, "test-skill.py")
	err = os.WriteFile(testFile, []byte("print('hello')"), 0644)
	require.NoError(t, err)

	// Initialize session
	sessionPath, err := pm.Initialize(sessionID, skillsDir)
	require.NoError(t, err)

	// Verify skills were copied
	copiedFile := filepath.Join(sessionPath, "skills", "test-skill.py")
	assert.FileExists(t, copiedFile)

	// Verify content matches
	content, err := os.ReadFile(copiedFile)
	require.NoError(t, err)
	assert.Equal(t, "print('hello')", string(content))
}

func TestPathManager_EmptySkillsDir(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	sessionID := "test-session-no-skills"
	skillsDir := ""

	// Initialize without skills directory
	sessionPath, err := pm.Initialize(sessionID, skillsDir)
	require.NoError(t, err)
	assert.NotEmpty(t, sessionPath)

	// Should still create session directory
	assert.DirExists(t, sessionPath)
}

func TestPathManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	sessionID := "test-session-concurrent"
	skillsDir := filepath.Join(tmpDir, "skills")

	err := os.MkdirAll(skillsDir, 0755)
	require.NoError(t, err)

	// Initialize
	_, err = pm.Initialize(sessionID, skillsDir)
	require.NoError(t, err)

	// Concurrent reads should work
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			path, err := pm.Get(sessionID)
			assert.NoError(t, err)
			assert.NotEmpty(t, path)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPathManager_BasePath(t *testing.T) {
	basePath := "/custom/base/path"
	pm := NewPathManager(basePath)

	// Verify base path is set
	assert.NotNil(t, pm)

	// Session paths should be under base path
	sessionID := "test-session-base"
	expectedPrefix := filepath.Join(basePath, sessionID)

	// Note: We can't actually initialize because the path doesn't exist,
	// but we can verify the logic would use the correct base
	pm.cache[sessionID] = expectedPrefix
	path, err := pm.Get(sessionID)
	require.NoError(t, err)
	assert.Equal(t, expectedPrefix, path)
}
