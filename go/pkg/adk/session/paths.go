package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	apperrors "github.com/kagent-dev/kagent/go/pkg/app/errors"
)

// PathManager manages session-specific directory paths
type PathManager struct {
	basePath string
	cache    map[string]string
	mu       sync.RWMutex
}

// NewPathManager creates a new PathManager
func NewPathManager(basePath string) *PathManager {
	if basePath == "" {
		basePath = "/tmp/kagent"
	}
	return &PathManager{
		basePath: basePath,
		cache:    make(map[string]string),
	}
}

// Initialize creates the session directory structure and returns the path
func (m *PathManager) Initialize(sessionID, skillsDir string) (string, error) {
	// Check cache first
	m.mu.RLock()
	if path, ok := m.cache[sessionID]; ok {
		m.mu.RUnlock()
		return path, nil
	}
	m.mu.RUnlock()

	// Create session directory
	sessionPath := filepath.Join(m.basePath, sessionID)
	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		return "", apperrors.New(apperrors.ErrCodePathManagement,
			"failed to create session directory", err)
	}

	// Create uploads directory
	uploadsPath := filepath.Join(sessionPath, "uploads")
	if err := os.MkdirAll(uploadsPath, 0755); err != nil {
		return "", apperrors.New(apperrors.ErrCodePathManagement,
			"failed to create uploads directory", err)
	}

	// Create outputs directory
	outputsPath := filepath.Join(sessionPath, "outputs")
	if err := os.MkdirAll(outputsPath, 0755); err != nil {
		return "", apperrors.New(apperrors.ErrCodePathManagement,
			"failed to create outputs directory", err)
	}

	// Create symlink to skills directory if provided
	if skillsDir != "" {
		skillsLink := filepath.Join(sessionPath, "skills")
		// Remove existing symlink if it exists
		os.Remove(skillsLink)
		if err := os.Symlink(skillsDir, skillsLink); err != nil {
			// Non-fatal error - skills may not be available
			fmt.Fprintf(os.Stderr, "Warning: failed to create skills symlink: %v\n", err)
		}
	}

	// Cache the result
	m.mu.Lock()
	m.cache[sessionID] = sessionPath
	m.mu.Unlock()

	return sessionPath, nil
}

// Get returns the cached session path or initializes it
func (m *PathManager) Get(sessionID string) (string, error) {
	m.mu.RLock()
	if path, ok := m.cache[sessionID]; ok {
		m.mu.RUnlock()
		return path, nil
	}
	m.mu.RUnlock()

	// Initialize if not cached
	return m.Initialize(sessionID, "")
}

// Clear removes cached paths
func (m *PathManager) Clear(sessionID *string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionID != nil {
		delete(m.cache, *sessionID)
	} else {
		// Clear all
		m.cache = make(map[string]string)
	}
}

// GetUploadsDir returns the uploads directory for a session
func (m *PathManager) GetUploadsDir(sessionID string) (string, error) {
	sessionPath, err := m.Get(sessionID)
	if err != nil {
		return "", err
	}
	return filepath.Join(sessionPath, "uploads"), nil
}

// GetOutputsDir returns the outputs directory for a session
func (m *PathManager) GetOutputsDir(sessionID string) (string, error) {
	sessionPath, err := m.Get(sessionID)
	if err != nil {
		return "", err
	}
	return filepath.Join(sessionPath, "outputs"), nil
}

// GetSkillsDir returns the skills directory for a session
func (m *PathManager) GetSkillsDir(sessionID string) (string, error) {
	sessionPath, err := m.Get(sessionID)
	if err != nil {
		return "", err
	}
	return filepath.Join(sessionPath, "skills"), nil
}
