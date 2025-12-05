package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FileService manages temporary file storage and cleanup
type FileService struct {
	storagePath string
	mu          sync.RWMutex
	cleanupJobs map[string]*time.Timer
}

// NewFileService creates a new FileService instance
func NewFileService(storagePath string) (*FileService, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileService{
		storagePath: storagePath,
		cleanupJobs: make(map[string]*time.Timer),
	}, nil
}

// SaveUpload saves an uploaded file with a unique identifier
func (fs *FileService) SaveUpload(file []byte, fileId string) (string, error) {
	if fileId == "" {
		fileId = uuid.New().String()
	}

	filePath := filepath.Join(fs.storagePath, fileId)

	if err := os.WriteFile(filePath, file, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fileId, nil
}

// GetFile retrieves a file by its identifier
func (fs *FileService) GetFile(fileId string) ([]byte, error) {
	filePath := filepath.Join(fs.storagePath, fileId)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", fileId)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// DeleteFile removes a file by its identifier
func (fs *FileService) DeleteFile(fileId string) error {
	filePath := filepath.Join(fs.storagePath, fileId)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, not an error
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Cancel any scheduled cleanup for this file
	fs.mu.Lock()
	if timer, exists := fs.cleanupJobs[fileId]; exists {
		timer.Stop()
		delete(fs.cleanupJobs, fileId)
	}
	fs.mu.Unlock()

	return nil
}

// ScheduleCleanup schedules a file for deletion after the specified delay
func (fs *FileService) ScheduleCleanup(fileId string, delay time.Duration) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Cancel existing cleanup job if any
	if timer, exists := fs.cleanupJobs[fileId]; exists {
		timer.Stop()
	}

	// Schedule new cleanup
	timer := time.AfterFunc(delay, func() {
		if err := fs.DeleteFile(fileId); err != nil {
			// Log error but don't fail - file might already be deleted
			fmt.Printf("Cleanup error for file %s: %v\n", fileId, err)
		}

		fs.mu.Lock()
		delete(fs.cleanupJobs, fileId)
		fs.mu.Unlock()
	})

	fs.cleanupJobs[fileId] = timer
}

// GenerateFileID generates a new unique file identifier
func (fs *FileService) GenerateFileID() string {
	return uuid.New().String()
}
