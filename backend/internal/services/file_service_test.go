package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: virtual-fitcheck, Property 16: Unique file identifiers
// For any set of uploaded images, each image should receive a unique identifier,
// with no two uploads sharing the same ID.
// Validates: Requirements 6.1
func TestProperty_UniqueFileIdentifiers(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("all generated file IDs are unique", prop.ForAll(
		func(numFiles int) bool {
			// Create temporary storage directory
			tempDir := filepath.Join(os.TempDir(), "test-file-service-"+time.Now().Format("20060102150405"))
			defer os.RemoveAll(tempDir)

			fs, err := NewFileService(tempDir)
			if err != nil {
				t.Logf("Failed to create FileService: %v", err)
				return false
			}

			// Generate multiple file IDs
			fileIDs := make(map[string]bool)
			testData := []byte("test data")

			for i := 0; i < numFiles; i++ {
				fileID := fs.GenerateFileID()

				// Check if this ID already exists
				if fileIDs[fileID] {
					t.Logf("Duplicate file ID found: %s", fileID)
					return false
				}

				fileIDs[fileID] = true

				// Also test SaveUpload with empty fileId (should generate unique ID)
				savedID, err := fs.SaveUpload(testData, "")
				if err != nil {
					t.Logf("Failed to save upload: %v", err)
					return false
				}

				if fileIDs[savedID] {
					t.Logf("Duplicate file ID from SaveUpload: %s", savedID)
					return false
				}

				fileIDs[savedID] = true
			}

			// Verify we have the expected number of unique IDs
			expectedCount := numFiles * 2 // GenerateFileID + SaveUpload for each iteration
			if len(fileIDs) != expectedCount {
				t.Logf("Expected %d unique IDs, got %d", expectedCount, len(fileIDs))
				return false
			}

			return true
		},
		gen.IntRange(1, 50), // Test with 1 to 50 files
	))

	properties.TestingRun(t)
}

// Additional unit tests for FileService functionality
func TestFileService_SaveAndGetFile(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test-file-service-"+time.Now().Format("20060102150405"))
	defer os.RemoveAll(tempDir)

	fs, err := NewFileService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create FileService: %v", err)
	}

	testData := []byte("test file content")
	fileID, err := fs.SaveUpload(testData, "")
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	retrievedData, err := fs.GetFile(fileID)
	if err != nil {
		t.Fatalf("Failed to get file: %v", err)
	}

	if string(retrievedData) != string(testData) {
		t.Errorf("Retrieved data doesn't match. Expected: %s, Got: %s", testData, retrievedData)
	}
}

func TestFileService_DeleteFile(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test-file-service-"+time.Now().Format("20060102150405"))
	defer os.RemoveAll(tempDir)

	fs, err := NewFileService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create FileService: %v", err)
	}

	testData := []byte("test file content")
	fileID, err := fs.SaveUpload(testData, "")
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	err = fs.DeleteFile(fileID)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Try to get deleted file
	_, err = fs.GetFile(fileID)
	if err == nil {
		t.Error("Expected error when getting deleted file, got nil")
	}
}

func TestFileService_ScheduleCleanup(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "test-file-service-"+time.Now().Format("20060102150405"))
	defer os.RemoveAll(tempDir)

	fs, err := NewFileService(tempDir)
	if err != nil {
		t.Fatalf("Failed to create FileService: %v", err)
	}

	testData := []byte("test file content")
	fileID, err := fs.SaveUpload(testData, "")
	if err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	// Schedule cleanup in 100ms
	fs.ScheduleCleanup(fileID, 100*time.Millisecond)

	// File should exist immediately
	_, err = fs.GetFile(fileID)
	if err != nil {
		t.Errorf("File should exist immediately after scheduling cleanup: %v", err)
	}

	// Wait for cleanup to occur
	time.Sleep(200 * time.Millisecond)

	// File should be deleted now
	_, err = fs.GetFile(fileID)
	if err == nil {
		t.Error("File should be deleted after cleanup delay")
	}
}
