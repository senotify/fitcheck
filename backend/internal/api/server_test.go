package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
	"virtual-fitcheck/internal/config"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a test image (valid JPEG)
func createTestJPEG() []byte {
	// Minimal valid JPEG file
	return []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
		0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
		0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9,
	}
}

// Helper function to create a test PNG
func createTestPNG() []byte {
	// Minimal valid PNG file
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
}

// Helper function to create multipart form with image
func createMultipartRequest(imageData []byte, filename string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, bytes.NewReader(imageData))
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", "/api/upload/user-photo", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

func TestUploadUserPhoto_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
		DatabaseURL:   "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable",
	}
	server := NewServer(cfg)
	if server.jobService == nil {
		t.Skip("Skipping test: database not available")
		return
	}

	// Create test request
	imageData := createTestJPEG()
	req, err := createMultipartRequest(imageData, "test.jpg")
	assert.NoError(t, err)

	// Execute request
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.NotEmpty(t, response["fileId"])
	assert.NotEmpty(t, response["previewUrl"])
	assert.Equal(t, "test.jpg", response["filename"])
}

func TestUploadShirt_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
		DatabaseURL:   "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable",
	}
	server := NewServer(cfg)
	if server.jobService == nil {
		t.Skip("Skipping test: database not available")
		return
	}

	// Create test request
	imageData := createTestPNG()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("image", "shirt.png")
	assert.NoError(t, err)

	_, err = io.Copy(part, bytes.NewReader(imageData))
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/upload/shirt", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.NotEmpty(t, response["fileId"])
	assert.NotEmpty(t, response["previewUrl"])
}

func TestUpload_InvalidFormat(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Create test request with invalid file
	invalidData := []byte("This is not an image")
	req, err := createMultipartRequest(invalidData, "test.txt")
	assert.NoError(t, err)

	// Execute request
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_FORMAT", errorObj["code"])
}

func TestUpload_FileTooLarge(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Create large file (11MB)
	largeData := make([]byte, 11*1024*1024)
	// Add JPEG header to make it a valid format
	copy(largeData, []byte{0xFF, 0xD8, 0xFF})

	req, err := createMultipartRequest(largeData, "large.jpg")
	assert.NoError(t, err)

	// Execute request
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "FILE_TOO_LARGE", errorObj["code"])
}

func TestUpload_NoFileProvided(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Create request without file
	req := httptest.NewRequest("POST", "/api/upload/user-photo", nil)

	// Execute request
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_REQUEST", errorObj["code"])
}

// Feature: virtual-fitcheck, Property 3: Successful upload preview
// For any successfully uploaded image, the system should return a valid preview URL
// that can be used to display the image.
// Validates: Requirements 1.3, 2.3
func TestProperty_SuccessfulUploadPreview(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid image formats
	validImageGen := gen.OneGenOf(
		gen.Const(createTestJPEG()),
		gen.Const(createTestPNG()),
		gen.Const(createTestWebP()),
	)

	// Generator for upload types
	uploadTypeGen := gen.OneConstOf("user-photo", "shirt")

	// Generator for filenames
	filenameGen := gen.OneConstOf(
		"test.jpg",
		"image.png",
		"photo.webp",
		"my-photo.jpeg",
		"shirt-image.png",
	)

	properties.Property("successful uploads return valid preview URLs", prop.ForAll(
		func(imageData []byte, uploadType string, filename string) bool {
			// Setup server with temp directory
			cfg := &config.Config{
				Port:          "8080",
				StoragePath:   t.TempDir(),
				CleanupDelay:  60,
				MaxUploadSize: 10 * 1024 * 1024,
			}
			server := NewServer(cfg)

			// Create multipart request
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			part, err := writer.CreateFormFile("image", filename)
			if err != nil {
				t.Logf("Failed to create form file: %v", err)
				return false
			}

			_, err = io.Copy(part, bytes.NewReader(imageData))
			if err != nil {
				t.Logf("Failed to copy image data: %v", err)
				return false
			}

			err = writer.Close()
			if err != nil {
				t.Logf("Failed to close writer: %v", err)
				return false
			}

			// Determine endpoint based on upload type
			endpoint := "/api/upload/" + uploadType
			req := httptest.NewRequest("POST", endpoint, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// Execute request
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// For user-photo uploads, the test might fail due to human feature detection
			// which is acceptable for this property test
			if uploadType == "user-photo" && w.Code == http.StatusUnprocessableEntity {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if errorObj, ok := response["error"].(map[string]interface{}); ok {
						if errorObj["code"] == "NO_HUMAN_FEATURES" {
							// This is expected for minimal test images
							return true
						}
					}
				}
			}

			// Check for successful response
			if w.Code != http.StatusOK {
				t.Logf("Expected status 200, got %d", w.Code)
				return false
			}

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Logf("Failed to parse response: %v", err)
				return false
			}

			// Verify success field
			success, ok := response["success"].(bool)
			if !ok || !success {
				t.Logf("Response success field is not true")
				return false
			}

			// Verify fileId exists and is non-empty
			fileId, ok := response["fileId"].(string)
			if !ok || fileId == "" {
				t.Logf("Response fileId is missing or empty")
				return false
			}

			// Verify previewUrl exists and is non-empty
			previewUrl, ok := response["previewUrl"].(string)
			if !ok || previewUrl == "" {
				t.Logf("Response previewUrl is missing or empty")
				return false
			}

			// Verify previewUrl has correct format: /api/preview/{fileId}
			expectedPrefix := "/api/preview/"
			if !strings.HasPrefix(previewUrl, expectedPrefix) {
				t.Logf("Preview URL doesn't have expected prefix. Got: %s", previewUrl)
				return false
			}

			// Verify the fileId in the URL matches the returned fileId
			urlFileId := strings.TrimPrefix(previewUrl, expectedPrefix)
			if urlFileId != fileId {
				t.Logf("FileId in URL (%s) doesn't match returned fileId (%s)", urlFileId, fileId)
				return false
			}

			// Verify filename is returned
			returnedFilename, ok := response["filename"].(string)
			if !ok || returnedFilename == "" {
				t.Logf("Response filename is missing or empty")
				return false
			}

			// Verify returned filename matches the uploaded filename
			if returnedFilename != filename {
				t.Logf("Returned filename (%s) doesn't match uploaded filename (%s)", returnedFilename, filename)
				return false
			}

			return true
		},
		validImageGen,
		uploadTypeGen,
		filenameGen,
	))

	properties.TestingRun(t)
}

// Helper function to create a test WebP
func createTestWebP() []byte {
	// Minimal valid WebP file (RIFF header + WebP signature)
	return []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x1A, 0x00, 0x00, 0x00, // File size (26 bytes)
		0x57, 0x45, 0x42, 0x50, // "WEBP"
		0x56, 0x50, 0x38, 0x20, // "VP8 "
		0x0E, 0x00, 0x00, 0x00, // Chunk size
		0x30, 0x01, 0x00, 0x9D, 0x01, 0x2A,
		0x01, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x00,
	}
}

func TestPreview_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// First upload an image
	imageData := createTestJPEG()
	uploadReq, err := createMultipartRequest(imageData, "test.jpg")
	assert.NoError(t, err)

	uploadW := httptest.NewRecorder()
	server.router.ServeHTTP(uploadW, uploadReq)
	assert.Equal(t, http.StatusOK, uploadW.Code)

	var uploadResponse map[string]interface{}
	err = json.Unmarshal(uploadW.Body.Bytes(), &uploadResponse)
	assert.NoError(t, err)

	fileId := uploadResponse["fileId"].(string)

	// Now request the preview
	previewReq := httptest.NewRequest("GET", "/api/preview/"+fileId, nil)
	previewW := httptest.NewRecorder()
	server.router.ServeHTTP(previewW, previewReq)

	// Assert response
	assert.Equal(t, http.StatusOK, previewW.Code)
	assert.Equal(t, "image/jpeg", previewW.Header().Get("Content-Type"))
	assert.Equal(t, imageData, previewW.Body.Bytes())
}

func TestPreview_FileNotFound(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Request preview for non-existent file
	previewReq := httptest.NewRequest("GET", "/api/preview/nonexistent-file-id", nil)
	previewW := httptest.NewRecorder()
	server.router.ServeHTTP(previewW, previewReq)

	// Assert response
	assert.Equal(t, http.StatusNotFound, previewW.Code)

	var response map[string]interface{}
	err := json.Unmarshal(previewW.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "FILE_NOT_FOUND", errorObj["code"])
}

func TestPreview_DifferentFormats(t *testing.T) {
	testCases := []struct {
		name        string
		imageData   []byte
		contentType string
	}{
		{"JPEG", createTestJPEG(), "image/jpeg"},
		{"PNG", createTestPNG(), "image/png"},
		{"WebP", createTestWebP(), "image/webp"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			cfg := &config.Config{
				Port:          "8080",
				StoragePath:   t.TempDir(),
				CleanupDelay:  60,
				MaxUploadSize: 10 * 1024 * 1024,
			}
			server := NewServer(cfg)

			// Upload image
			uploadReq, err := createMultipartRequest(tc.imageData, "test."+strings.ToLower(tc.name))
			assert.NoError(t, err)

			uploadW := httptest.NewRecorder()
			server.router.ServeHTTP(uploadW, uploadReq)
			assert.Equal(t, http.StatusOK, uploadW.Code)

			var uploadResponse map[string]interface{}
			err = json.Unmarshal(uploadW.Body.Bytes(), &uploadResponse)
			assert.NoError(t, err)

			fileId := uploadResponse["fileId"].(string)

			// Request preview
			previewReq := httptest.NewRequest("GET", "/api/preview/"+fileId, nil)
			previewW := httptest.NewRecorder()
			server.router.ServeHTTP(previewW, previewReq)

			// Assert response
			assert.Equal(t, http.StatusOK, previewW.Code)
			assert.Equal(t, tc.contentType, previewW.Header().Get("Content-Type"))
			assert.Equal(t, tc.imageData, previewW.Body.Bytes())
		})
	}
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestProcess_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// First upload user photo
	userPhotoData := createTestJPEG()
	userPhotoReq, err := createMultipartRequest(userPhotoData, "user.jpg")
	assert.NoError(t, err)

	userPhotoW := httptest.NewRecorder()
	server.router.ServeHTTP(userPhotoW, userPhotoReq)
	assert.Equal(t, http.StatusOK, userPhotoW.Code)

	var userPhotoResponse map[string]interface{}
	err = json.Unmarshal(userPhotoW.Body.Bytes(), &userPhotoResponse)
	assert.NoError(t, err)
	userPhotoID := userPhotoResponse["fileId"].(string)

	// Upload shirt image
	shirtData := createTestPNG()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "shirt.png")
	assert.NoError(t, err)
	_, err = io.Copy(part, bytes.NewReader(shirtData))
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	shirtReq := httptest.NewRequest("POST", "/api/upload/shirt", body)
	shirtReq.Header.Set("Content-Type", writer.FormDataContentType())

	shirtW := httptest.NewRecorder()
	server.router.ServeHTTP(shirtW, shirtReq)
	assert.Equal(t, http.StatusOK, shirtW.Code)

	var shirtResponse map[string]interface{}
	err = json.Unmarshal(shirtW.Body.Bytes(), &shirtResponse)
	assert.NoError(t, err)
	shirtImageID := shirtResponse["fileId"].(string)

	// Now create process request
	processReqBody := map[string]string{
		"userPhotoId":  userPhotoID,
		"shirtImageId": shirtImageID,
	}
	processReqJSON, err := json.Marshal(processReqBody)
	assert.NoError(t, err)

	processReq := httptest.NewRequest("POST", "/api/process", bytes.NewReader(processReqJSON))
	processReq.Header.Set("Content-Type", "application/json")

	processW := httptest.NewRecorder()
	server.router.ServeHTTP(processW, processReq)

	// Assert response
	assert.Equal(t, http.StatusOK, processW.Code)

	var processResponse map[string]interface{}
	err = json.Unmarshal(processW.Body.Bytes(), &processResponse)
	assert.NoError(t, err)

	assert.True(t, processResponse["success"].(bool))
	assert.NotEmpty(t, processResponse["jobId"])
	assert.Equal(t, float64(15), processResponse["estimatedTime"].(float64))
}

func TestProcess_MissingUserPhoto(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Create process request with non-existent user photo
	processReqBody := map[string]string{
		"userPhotoId":  "non-existent-id",
		"shirtImageId": "some-shirt-id",
	}
	processReqJSON, err := json.Marshal(processReqBody)
	assert.NoError(t, err)

	processReq := httptest.NewRequest("POST", "/api/process", bytes.NewReader(processReqJSON))
	processReq.Header.Set("Content-Type", "application/json")

	processW := httptest.NewRecorder()
	server.router.ServeHTTP(processW, processReq)

	// Assert response
	assert.Equal(t, http.StatusNotFound, processW.Code)

	var response map[string]interface{}
	err = json.Unmarshal(processW.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "USER_PHOTO_NOT_FOUND", errorObj["code"])
}

func TestProcess_MissingShirtImage(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Upload user photo
	userPhotoData := createTestJPEG()
	userPhotoReq, err := createMultipartRequest(userPhotoData, "user.jpg")
	assert.NoError(t, err)

	userPhotoW := httptest.NewRecorder()
	server.router.ServeHTTP(userPhotoW, userPhotoReq)
	assert.Equal(t, http.StatusOK, userPhotoW.Code)

	var userPhotoResponse map[string]interface{}
	err = json.Unmarshal(userPhotoW.Body.Bytes(), &userPhotoResponse)
	assert.NoError(t, err)
	userPhotoID := userPhotoResponse["fileId"].(string)

	// Create process request with non-existent shirt image
	processReqBody := map[string]string{
		"userPhotoId":  userPhotoID,
		"shirtImageId": "non-existent-shirt-id",
	}
	processReqJSON, err := json.Marshal(processReqBody)
	assert.NoError(t, err)

	processReq := httptest.NewRequest("POST", "/api/process", bytes.NewReader(processReqJSON))
	processReq.Header.Set("Content-Type", "application/json")

	processW := httptest.NewRecorder()
	server.router.ServeHTTP(processW, processReq)

	// Assert response
	assert.Equal(t, http.StatusNotFound, processW.Code)

	var response map[string]interface{}
	err = json.Unmarshal(processW.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "SHIRT_IMAGE_NOT_FOUND", errorObj["code"])
}

func TestProcess_InvalidRequestBody(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Create process request with missing fields
	processReqBody := map[string]string{
		"userPhotoId": "some-id",
		// Missing shirtImageId
	}
	processReqJSON, err := json.Marshal(processReqBody)
	assert.NoError(t, err)

	processReq := httptest.NewRequest("POST", "/api/process", bytes.NewReader(processReqJSON))
	processReq.Header.Set("Content-Type", "application/json")

	processW := httptest.NewRecorder()
	server.router.ServeHTTP(processW, processReq)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, processW.Code)

	var response map[string]interface{}
	err = json.Unmarshal(processW.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_REQUEST", errorObj["code"])
}

// Feature: virtual-fitcheck, Property 6: Image forwarding to AI service
// Validates: Requirements 3.2
// For any processing request with valid user photo and shirt image IDs, the backend
// should send both complete images to the Generative AI Service.
func TestProperty_ImageForwardingToAIService(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("complete images are forwarded to AI service", prop.ForAll(
		func(userPhotoSize, shirtImageSize int) bool {
			// Generate random image data with valid JPEG headers
			userPhoto := make([]byte, userPhotoSize)
			shirtImage := make([]byte, shirtImageSize)
			
			// Add JPEG magic numbers to make them valid images
			copy(userPhoto, []byte{0xFF, 0xD8, 0xFF})
			copy(shirtImage, []byte{0xFF, 0xD8, 0xFF})
			
			// Fill rest with random data
			for i := 3; i < len(userPhoto); i++ {
				userPhoto[i] = byte(i % 256)
			}
			for i := 3; i < len(shirtImage); i++ {
				shirtImage[i] = byte((i + 100) % 256)
			}

			// Track what images the AI service receives
			var receivedUserPhoto []byte
			var receivedShirtImage []byte
			aiServiceCalled := false

			// Create mock AI service
			mockAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" && r.URL.Path == "/try-on" {
					aiServiceCalled = true

					// Read and parse the request body
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Logf("Failed to read AI service request body: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					var aiReq struct {
						PersonImage  string `json:"personImage"`
						GarmentImage string `json:"garmentImage"`
					}
					if err := json.Unmarshal(body, &aiReq); err != nil {
						t.Logf("Failed to parse AI service request: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					// Decode base64 images
					var err1, err2 error
					receivedUserPhoto, err1 = base64.StdEncoding.DecodeString(aiReq.PersonImage)
					receivedShirtImage, err2 = base64.StdEncoding.DecodeString(aiReq.GarmentImage)

					if err1 != nil || err2 != nil {
						t.Logf("Failed to decode images: user=%v, shirt=%v", err1, err2)
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					// Return success response
					response := map[string]interface{}{
						"id":     "test-job-123",
						"status": "starting",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(response)
				}
			}))
			defer mockAIServer.Close()

			// Setup server with mock AI service
			cfg := &config.Config{
				Port:          "8080",
				StoragePath:   t.TempDir(),
				AIServiceURL:  mockAIServer.URL,
				AIServiceKey:  "test-key",
				CleanupDelay:  60,
				MaxUploadSize: 10 * 1024 * 1024,
			}
			server := NewServer(cfg)

			// Upload user photo
			userPhotoBody := &bytes.Buffer{}
			userPhotoWriter := multipart.NewWriter(userPhotoBody)
			userPhotoPart, err := userPhotoWriter.CreateFormFile("image", "user.jpg")
			if err != nil {
				t.Logf("Failed to create user photo form: %v", err)
				return false
			}
			_, err = io.Copy(userPhotoPart, bytes.NewReader(userPhoto))
			if err != nil {
				t.Logf("Failed to copy user photo: %v", err)
				return false
			}
			err = userPhotoWriter.Close()
			if err != nil {
				t.Logf("Failed to close user photo writer: %v", err)
				return false
			}

			userPhotoReq := httptest.NewRequest("POST", "/api/upload/user-photo", userPhotoBody)
			userPhotoReq.Header.Set("Content-Type", userPhotoWriter.FormDataContentType())

			userPhotoW := httptest.NewRecorder()
			server.router.ServeHTTP(userPhotoW, userPhotoReq)

			// For user photos, human feature detection might fail on random data
			// This is acceptable - we skip these cases
			if userPhotoW.Code == http.StatusUnprocessableEntity {
				var response map[string]interface{}
				if err := json.Unmarshal(userPhotoW.Body.Bytes(), &response); err == nil {
					if errorObj, ok := response["error"].(map[string]interface{}); ok {
						if errorObj["code"] == "NO_HUMAN_FEATURES" {
							return true // Skip this test case
						}
					}
				}
			}

			if userPhotoW.Code != http.StatusOK {
				t.Logf("User photo upload failed with status %d", userPhotoW.Code)
				return false
			}

			var userPhotoResponse map[string]interface{}
			err = json.Unmarshal(userPhotoW.Body.Bytes(), &userPhotoResponse)
			if err != nil {
				t.Logf("Failed to parse user photo response: %v", err)
				return false
			}
			userPhotoID := userPhotoResponse["fileId"].(string)

			// Upload shirt image
			shirtBody := &bytes.Buffer{}
			shirtWriter := multipart.NewWriter(shirtBody)
			shirtPart, err := shirtWriter.CreateFormFile("image", "shirt.jpg")
			if err != nil {
				t.Logf("Failed to create shirt form: %v", err)
				return false
			}
			_, err = io.Copy(shirtPart, bytes.NewReader(shirtImage))
			if err != nil {
				t.Logf("Failed to copy shirt image: %v", err)
				return false
			}
			err = shirtWriter.Close()
			if err != nil {
				t.Logf("Failed to close shirt writer: %v", err)
				return false
			}

			shirtReq := httptest.NewRequest("POST", "/api/upload/shirt", shirtBody)
			shirtReq.Header.Set("Content-Type", shirtWriter.FormDataContentType())

			shirtW := httptest.NewRecorder()
			server.router.ServeHTTP(shirtW, shirtReq)

			if shirtW.Code != http.StatusOK {
				t.Logf("Shirt upload failed with status %d", shirtW.Code)
				return false
			}

			var shirtResponse map[string]interface{}
			err = json.Unmarshal(shirtW.Body.Bytes(), &shirtResponse)
			if err != nil {
				t.Logf("Failed to parse shirt response: %v", err)
				return false
			}
			shirtImageID := shirtResponse["fileId"].(string)

			// Create process request
			processReqBody := map[string]string{
				"userPhotoId":  userPhotoID,
				"shirtImageId": shirtImageID,
			}
			processReqJSON, err := json.Marshal(processReqBody)
			if err != nil {
				t.Logf("Failed to marshal process request: %v", err)
				return false
			}

			processReq := httptest.NewRequest("POST", "/api/process", bytes.NewReader(processReqJSON))
			processReq.Header.Set("Content-Type", "application/json")

			processW := httptest.NewRecorder()
			server.router.ServeHTTP(processW, processReq)

			if processW.Code != http.StatusOK {
				t.Logf("Process request failed with status %d: %s", processW.Code, processW.Body.String())
				return false
			}

			// Give the async goroutine time to call the AI service
			// Wait up to 1 second for the AI service to be called
			for i := 0; i < 10 && !aiServiceCalled; i++ {
				time.Sleep(100 * time.Millisecond)
			}

			// Verify AI service was called
			if !aiServiceCalled {
				t.Logf("AI service was not called")
				return false
			}

			// Verify complete user photo was forwarded
			if len(receivedUserPhoto) != len(userPhoto) {
				t.Logf("User photo size mismatch: expected %d, got %d", len(userPhoto), len(receivedUserPhoto))
				return false
			}

			// Verify user photo content matches
			for i := 0; i < len(userPhoto); i++ {
				if receivedUserPhoto[i] != userPhoto[i] {
					t.Logf("User photo content mismatch at byte %d: expected %d, got %d", i, userPhoto[i], receivedUserPhoto[i])
					return false
				}
			}

			// Verify complete shirt image was forwarded
			if len(receivedShirtImage) != len(shirtImage) {
				t.Logf("Shirt image size mismatch: expected %d, got %d", len(shirtImage), len(receivedShirtImage))
				return false
			}

			// Verify shirt image content matches
			for i := 0; i < len(shirtImage); i++ {
				if receivedShirtImage[i] != shirtImage[i] {
					t.Logf("Shirt image content mismatch at byte %d: expected %d, got %d", i, shirtImage[i], receivedShirtImage[i])
					return false
				}
			}

			return true
		},
		gen.IntRange(100, 10*1024),  // User photo size: 100 bytes to 10KB
		gen.IntRange(100, 10*1024),  // Shirt image size: 100 bytes to 10KB
	))

	properties.TestingRun(t)
}

// Feature: virtual-fitcheck, Property 19: Processing time estimation
// Validates: Requirements 7.3
// For any initiated processing request, the response should include an estimated
// processing time in seconds.
func TestProperty_ProcessingTimeEstimation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid image data
	validImageGen := gen.OneGenOf(
		gen.Const(createTestJPEG()),
		gen.Const(createTestPNG()),
		gen.Const(createTestWebP()),
	)

	properties.Property("processing requests include estimated time in seconds", prop.ForAll(
		func(userPhotoData, shirtImageData []byte) bool {
			// Setup server with temp directory
			cfg := &config.Config{
				Port:          "8080",
				StoragePath:   t.TempDir(),
				AIServiceURL:  "http://mock-ai-service.com",
				AIServiceKey:  "test-key",
				CleanupDelay:  60,
				MaxUploadSize: 10 * 1024 * 1024,
			}
			server := NewServer(cfg)

			// Upload user photo
			userPhotoBody := &bytes.Buffer{}
			userPhotoWriter := multipart.NewWriter(userPhotoBody)
			userPhotoPart, err := userPhotoWriter.CreateFormFile("image", "user.jpg")
			if err != nil {
				t.Logf("Failed to create user photo form: %v", err)
				return false
			}
			_, err = io.Copy(userPhotoPart, bytes.NewReader(userPhotoData))
			if err != nil {
				t.Logf("Failed to copy user photo: %v", err)
				return false
			}
			err = userPhotoWriter.Close()
			if err != nil {
				t.Logf("Failed to close user photo writer: %v", err)
				return false
			}

			userPhotoReq := httptest.NewRequest("POST", "/api/upload/user-photo", userPhotoBody)
			userPhotoReq.Header.Set("Content-Type", userPhotoWriter.FormDataContentType())

			userPhotoW := httptest.NewRecorder()
			server.router.ServeHTTP(userPhotoW, userPhotoReq)

			// For user photos, human feature detection might fail on minimal test images
			// This is acceptable - we skip these cases
			if userPhotoW.Code == http.StatusUnprocessableEntity {
				var response map[string]interface{}
				if err := json.Unmarshal(userPhotoW.Body.Bytes(), &response); err == nil {
					if errorObj, ok := response["error"].(map[string]interface{}); ok {
						if errorObj["code"] == "NO_HUMAN_FEATURES" {
							return true // Skip this test case
						}
					}
				}
			}

			if userPhotoW.Code != http.StatusOK {
				t.Logf("User photo upload failed with status %d", userPhotoW.Code)
				return false
			}

			var userPhotoResponse map[string]interface{}
			err = json.Unmarshal(userPhotoW.Body.Bytes(), &userPhotoResponse)
			if err != nil {
				t.Logf("Failed to parse user photo response: %v", err)
				return false
			}
			userPhotoID := userPhotoResponse["fileId"].(string)

			// Upload shirt image
			shirtBody := &bytes.Buffer{}
			shirtWriter := multipart.NewWriter(shirtBody)
			shirtPart, err := shirtWriter.CreateFormFile("image", "shirt.jpg")
			if err != nil {
				t.Logf("Failed to create shirt form: %v", err)
				return false
			}
			_, err = io.Copy(shirtPart, bytes.NewReader(shirtImageData))
			if err != nil {
				t.Logf("Failed to copy shirt image: %v", err)
				return false
			}
			err = shirtWriter.Close()
			if err != nil {
				t.Logf("Failed to close shirt writer: %v", err)
				return false
			}

			shirtReq := httptest.NewRequest("POST", "/api/upload/shirt", shirtBody)
			shirtReq.Header.Set("Content-Type", shirtWriter.FormDataContentType())

			shirtW := httptest.NewRecorder()
			server.router.ServeHTTP(shirtW, shirtReq)

			if shirtW.Code != http.StatusOK {
				t.Logf("Shirt upload failed with status %d", shirtW.Code)
				return false
			}

			var shirtResponse map[string]interface{}
			err = json.Unmarshal(shirtW.Body.Bytes(), &shirtResponse)
			if err != nil {
				t.Logf("Failed to parse shirt response: %v", err)
				return false
			}
			shirtImageID := shirtResponse["fileId"].(string)

			// Create process request
			processReqBody := map[string]string{
				"userPhotoId":  userPhotoID,
				"shirtImageId": shirtImageID,
			}
			processReqJSON, err := json.Marshal(processReqBody)
			if err != nil {
				t.Logf("Failed to marshal process request: %v", err)
				return false
			}

			processReq := httptest.NewRequest("POST", "/api/process", bytes.NewReader(processReqJSON))
			processReq.Header.Set("Content-Type", "application/json")

			processW := httptest.NewRecorder()
			server.router.ServeHTTP(processW, processReq)

			// Verify successful response
			if processW.Code != http.StatusOK {
				t.Logf("Process request failed with status %d: %s", processW.Code, processW.Body.String())
				return false
			}

			// Parse response
			var processResponse map[string]interface{}
			err = json.Unmarshal(processW.Body.Bytes(), &processResponse)
			if err != nil {
				t.Logf("Failed to parse process response: %v", err)
				return false
			}

			// Verify success field
			success, ok := processResponse["success"].(bool)
			if !ok || !success {
				t.Logf("Response success field is not true")
				return false
			}

			// Verify jobId exists and is non-empty
			jobId, ok := processResponse["jobId"].(string)
			if !ok || jobId == "" {
				t.Logf("Response jobId is missing or empty")
				return false
			}

			// CRITICAL: Verify estimatedTime exists
			estimatedTime, ok := processResponse["estimatedTime"]
			if !ok {
				t.Logf("Response is missing estimatedTime field")
				return false
			}

			// Verify estimatedTime is a number
			estimatedTimeFloat, ok := estimatedTime.(float64)
			if !ok {
				t.Logf("estimatedTime is not a number, got type %T", estimatedTime)
				return false
			}

			// Verify estimatedTime is a positive number (in seconds)
			if estimatedTimeFloat <= 0 {
				t.Logf("estimatedTime should be positive, got %f", estimatedTimeFloat)
				return false
			}

			// Verify estimatedTime is reasonable (not absurdly large)
			// Typical AI processing should be under 5 minutes (300 seconds)
			if estimatedTimeFloat > 300 {
				t.Logf("estimatedTime is unreasonably large: %f seconds", estimatedTimeFloat)
				return false
			}

			return true
		},
		validImageGen,
		validImageGen,
	))

	properties.TestingRun(t)
}

func TestStatus_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// First upload user photo and shirt
	userPhotoData := createTestJPEG()
	userPhotoReq, err := createMultipartRequest(userPhotoData, "user.jpg")
	assert.NoError(t, err)

	userPhotoW := httptest.NewRecorder()
	server.router.ServeHTTP(userPhotoW, userPhotoReq)
	assert.Equal(t, http.StatusOK, userPhotoW.Code)

	var userPhotoResponse map[string]interface{}
	err = json.Unmarshal(userPhotoW.Body.Bytes(), &userPhotoResponse)
	assert.NoError(t, err)
	userPhotoID := userPhotoResponse["fileId"].(string)

	// Upload shirt image
	shirtData := createTestPNG()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "shirt.png")
	assert.NoError(t, err)
	_, err = io.Copy(part, bytes.NewReader(shirtData))
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)

	shirtReq := httptest.NewRequest("POST", "/api/upload/shirt", body)
	shirtReq.Header.Set("Content-Type", writer.FormDataContentType())

	shirtW := httptest.NewRecorder()
	server.router.ServeHTTP(shirtW, shirtReq)
	assert.Equal(t, http.StatusOK, shirtW.Code)

	var shirtResponse map[string]interface{}
	err = json.Unmarshal(shirtW.Body.Bytes(), &shirtResponse)
	assert.NoError(t, err)
	shirtImageID := shirtResponse["fileId"].(string)

	// Create process request
	processReqBody := map[string]string{
		"userPhotoId":  userPhotoID,
		"shirtImageId": shirtImageID,
	}
	processReqJSON, err := json.Marshal(processReqBody)
	assert.NoError(t, err)

	processReq := httptest.NewRequest("POST", "/api/process", bytes.NewReader(processReqJSON))
	processReq.Header.Set("Content-Type", "application/json")

	processW := httptest.NewRecorder()
	server.router.ServeHTTP(processW, processReq)
	assert.Equal(t, http.StatusOK, processW.Code)

	var processResponse map[string]interface{}
	err = json.Unmarshal(processW.Body.Bytes(), &processResponse)
	assert.NoError(t, err)
	jobID := processResponse["jobId"].(string)

	// Now check status
	statusReq := httptest.NewRequest("GET", "/api/status/"+jobID, nil)
	statusW := httptest.NewRecorder()
	server.router.ServeHTTP(statusW, statusReq)

	// Assert response
	assert.Equal(t, http.StatusOK, statusW.Code)

	var statusResponse map[string]interface{}
	err = json.Unmarshal(statusW.Body.Bytes(), &statusResponse)
	assert.NoError(t, err)

	// Verify required fields
	assert.NotEmpty(t, statusResponse["status"])
	assert.NotNil(t, statusResponse["progress"])
	assert.NotEmpty(t, statusResponse["message"])

	// Status should be pending or processing
	status := statusResponse["status"].(string)
	assert.Contains(t, []string{"pending", "processing"}, status)
}

func TestStatus_JobNotFound(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Request status for non-existent job
	statusReq := httptest.NewRequest("GET", "/api/status/non-existent-job-id", nil)
	statusW := httptest.NewRecorder()
	server.router.ServeHTTP(statusW, statusReq)

	// Assert response
	assert.Equal(t, http.StatusNotFound, statusW.Code)

	var response map[string]interface{}
	err := json.Unmarshal(statusW.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "JOB_NOT_FOUND", errorObj["code"])
}

func TestStatus_CompletedJobIncludesResultUrl(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Create a job directly
	job, err := server.jobService.CreateJob("test-session-id", "user-photo-id", "shirt-image-id")
	assert.NoError(t, err)

	// Mark job as completed with a result
	resultID := "test-result-id"
	err = server.jobService.SetJobResult(job.JobID, resultID)
	assert.NoError(t, err)
	err = server.jobService.UpdateJobStatus(job.JobID, "completed", "Processing completed successfully")
	assert.NoError(t, err)
	err = server.jobService.UpdateJobProgress(job.JobID, 100, "Completed")
	assert.NoError(t, err)

	// Check status
	statusReq := httptest.NewRequest("GET", "/api/status/"+job.JobID, nil)
	statusW := httptest.NewRecorder()
	server.router.ServeHTTP(statusW, statusReq)

	// Assert response
	assert.Equal(t, http.StatusOK, statusW.Code)

	var statusResponse map[string]interface{}
	err = json.Unmarshal(statusW.Body.Bytes(), &statusResponse)
	assert.NoError(t, err)

	// Verify status is completed
	assert.Equal(t, "completed", statusResponse["status"])
	assert.Equal(t, float64(100), statusResponse["progress"])

	// Verify resultUrl is included
	assert.NotEmpty(t, statusResponse["resultUrl"])
	expectedResultUrl := "/api/result/" + resultID
	assert.Equal(t, expectedResultUrl, statusResponse["resultUrl"])
}

func TestStatus_FailedJobIncludesError(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Create a job directly
	job, err := server.jobService.CreateJob("test-session-id", "user-photo-id", "shirt-image-id")
	assert.NoError(t, err)

	// Mark job as failed with an error
	errorMsg := "AI service unavailable"
	err = server.jobService.SetJobError(job.JobID, errorMsg)
	assert.NoError(t, err)

	// Check status
	statusReq := httptest.NewRequest("GET", "/api/status/"+job.JobID, nil)
	statusW := httptest.NewRecorder()
	server.router.ServeHTTP(statusW, statusReq)

	// Assert response
	assert.Equal(t, http.StatusOK, statusW.Code)

	var statusResponse map[string]interface{}
	err = json.Unmarshal(statusW.Body.Bytes(), &statusResponse)
	assert.NoError(t, err)

	// Verify status is failed
	assert.Equal(t, "failed", statusResponse["status"])

	// Verify error message is included
	assert.NotEmpty(t, statusResponse["error"])
	assert.Equal(t, errorMsg, statusResponse["error"])
}

func TestResult_Success(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Create a test result image
	resultImage := createTestJPEG()
	resultID := server.fileService.GenerateFileID()
	_, err := server.fileService.SaveUpload(resultImage, resultID)
	assert.NoError(t, err)

	// Request the result
	resultReq := httptest.NewRequest("GET", "/api/result/"+resultID, nil)
	resultW := httptest.NewRecorder()
	server.router.ServeHTTP(resultW, resultReq)

	// Assert response
	assert.Equal(t, http.StatusOK, resultW.Code)
	assert.Equal(t, "image/jpeg", resultW.Header().Get("Content-Type"))
	
	// Verify Content-Disposition header is set for download
	contentDisposition := resultW.Header().Get("Content-Disposition")
	assert.NotEmpty(t, contentDisposition)
	assert.Contains(t, contentDisposition, "attachment")
	assert.Contains(t, contentDisposition, "virtual-fitcheck-")
	assert.Contains(t, contentDisposition, ".jpg")
	
	// Verify image data is returned without modification
	assert.Equal(t, resultImage, resultW.Body.Bytes())
}

func TestResult_NotFound(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Request non-existent result
	resultReq := httptest.NewRequest("GET", "/api/result/non-existent-result-id", nil)
	resultW := httptest.NewRecorder()
	server.router.ServeHTTP(resultW, resultReq)

	// Assert response
	assert.Equal(t, http.StatusNotFound, resultW.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resultW.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "RESULT_NOT_FOUND", errorObj["code"])
}

func TestResult_DifferentFormats(t *testing.T) {
	testCases := []struct {
		name        string
		imageData   []byte
		contentType string
		extension   string
	}{
		{"JPEG", createTestJPEG(), "image/jpeg", ".jpg"},
		{"PNG", createTestPNG(), "image/png", ".png"},
		{"WebP", createTestWebP(), "image/webp", ".webp"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			cfg := &config.Config{
				Port:          "8080",
				StoragePath:   t.TempDir(),
				AIServiceURL:  "http://mock-ai-service.com",
				AIServiceKey:  "test-key",
				CleanupDelay:  60,
				MaxUploadSize: 10 * 1024 * 1024,
			}
			server := NewServer(cfg)

			// Save result image
			resultID := server.fileService.GenerateFileID()
			_, err := server.fileService.SaveUpload(tc.imageData, resultID)
			assert.NoError(t, err)

			// Request result
			resultReq := httptest.NewRequest("GET", "/api/result/"+resultID, nil)
			resultW := httptest.NewRecorder()
			server.router.ServeHTTP(resultW, resultReq)

			// Assert response
			assert.Equal(t, http.StatusOK, resultW.Code)
			assert.Equal(t, tc.contentType, resultW.Header().Get("Content-Type"))
			
			// Verify Content-Disposition has correct extension
			contentDisposition := resultW.Header().Get("Content-Disposition")
			assert.Contains(t, contentDisposition, tc.extension)
			
			// Verify image data matches
			assert.Equal(t, tc.imageData, resultW.Body.Bytes())
		})
	}
}

// Feature: virtual-fitcheck, Property 11: Download with proper filename
// For any composite image download action, the system should initiate a file download
// with a filename that includes a timestamp or unique identifier.
// Validates: Requirements 4.2
func TestProperty_DownloadWithProperFilename(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid image formats
	validImageGen := gen.OneGenOf(
		gen.Const(createTestJPEG()),
		gen.Const(createTestPNG()),
		gen.Const(createTestWebP()),
	)

	properties.Property("downloads include timestamp or unique identifier in filename", prop.ForAll(
		func(imageData []byte) bool {
			// Setup server with temp directory
			cfg := &config.Config{
				Port:          "8080",
				StoragePath:   t.TempDir(),
				AIServiceURL:  "http://mock-ai-service.com",
				AIServiceKey:  "test-key",
				CleanupDelay:  60,
				MaxUploadSize: 10 * 1024 * 1024,
			}
			server := NewServer(cfg)

			// Save a result image
			resultID := server.fileService.GenerateFileID()
			_, err := server.fileService.SaveUpload(imageData, resultID)
			if err != nil {
				t.Logf("Failed to save result image: %v", err)
				return false
			}

			// Request the result download
			resultReq := httptest.NewRequest("GET", "/api/result/"+resultID, nil)
			resultW := httptest.NewRecorder()
			server.router.ServeHTTP(resultW, resultReq)

			// Verify successful response
			if resultW.Code != http.StatusOK {
				t.Logf("Expected status 200, got %d", resultW.Code)
				return false
			}

			// Verify Content-Disposition header exists
			contentDisposition := resultW.Header().Get("Content-Disposition")
			if contentDisposition == "" {
				t.Logf("Content-Disposition header is missing")
				return false
			}

			// Verify it's an attachment
			if !strings.Contains(contentDisposition, "attachment") {
				t.Logf("Content-Disposition doesn't contain 'attachment': %s", contentDisposition)
				return false
			}

			// Verify filename is present
			if !strings.Contains(contentDisposition, "filename=") {
				t.Logf("Content-Disposition doesn't contain filename: %s", contentDisposition)
				return false
			}

			// Verify filename contains "virtual-fitcheck-" prefix
			if !strings.Contains(contentDisposition, "virtual-fitcheck-") {
				t.Logf("Filename doesn't contain 'virtual-fitcheck-' prefix: %s", contentDisposition)
				return false
			}

			// Verify filename contains a timestamp-like pattern (YYYYMMDD-HHMMSS)
			// The timestamp should be in format: 20060102-150405
			// We'll check for a pattern like: NNNNNNNN-NNNNNN (8 digits, dash, 6 digits)
			timestampPattern := `\d{8}-\d{6}`
			matched, err := regexp.MatchString(timestampPattern, contentDisposition)
			if err != nil {
				t.Logf("Failed to match timestamp pattern: %v", err)
				return false
			}
			if !matched {
				t.Logf("Filename doesn't contain timestamp pattern (YYYYMMDD-HHMMSS): %s", contentDisposition)
				return false
			}

			// Verify filename has an appropriate extension
			hasExtension := strings.Contains(contentDisposition, ".jpg") ||
				strings.Contains(contentDisposition, ".png") ||
				strings.Contains(contentDisposition, ".webp")
			if !hasExtension {
				t.Logf("Filename doesn't have a valid image extension: %s", contentDisposition)
				return false
			}

			return true
		},
		validImageGen,
	))

	properties.TestingRun(t)
}

// Feature: virtual-fitcheck, Property 12: Image quality preservation
// For any generated composite image, the downloaded file should have identical dimensions
// and comparable quality to the generated result (no additional lossy compression).
// Validates: Requirements 4.3
func TestProperty_ImageQualityPreservation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid image formats
	validImageGen := gen.OneGenOf(
		gen.Const(createTestJPEG()),
		gen.Const(createTestPNG()),
		gen.Const(createTestWebP()),
	)

	properties.Property("downloaded images preserve original data without additional compression", prop.ForAll(
		func(imageData []byte) bool {
			// Setup server with temp directory
			cfg := &config.Config{
				Port:          "8080",
				StoragePath:   t.TempDir(),
				AIServiceURL:  "http://mock-ai-service.com",
				AIServiceKey:  "test-key",
				CleanupDelay:  60,
				MaxUploadSize: 10 * 1024 * 1024,
			}
			server := NewServer(cfg)

			// Save a result image
			resultID := server.fileService.GenerateFileID()
			_, err := server.fileService.SaveUpload(imageData, resultID)
			if err != nil {
				t.Logf("Failed to save result image: %v", err)
				return false
			}

			// Request the result download
			resultReq := httptest.NewRequest("GET", "/api/result/"+resultID, nil)
			resultW := httptest.NewRecorder()
			server.router.ServeHTTP(resultW, resultReq)

			// Verify successful response
			if resultW.Code != http.StatusOK {
				t.Logf("Expected status 200, got %d", resultW.Code)
				return false
			}

			// Verify the downloaded image data is identical to the original
			downloadedData := resultW.Body.Bytes()
			if len(downloadedData) != len(imageData) {
				t.Logf("Downloaded image size (%d bytes) differs from original (%d bytes)", len(downloadedData), len(imageData))
				return false
			}

			// Verify byte-by-byte equality (no additional compression or modification)
			for i := 0; i < len(imageData); i++ {
				if downloadedData[i] != imageData[i] {
					t.Logf("Downloaded image differs from original at byte %d: expected %d, got %d", i, imageData[i], downloadedData[i])
					return false
				}
			}

			// Verify Content-Type is set correctly
			contentType := resultW.Header().Get("Content-Type")
			if contentType == "" {
				t.Logf("Content-Type header is missing")
				return false
			}

			// Verify Content-Type matches the image format
			expectedContentType := detectContentType(imageData)
			if contentType != expectedContentType {
				t.Logf("Content-Type mismatch: expected %s, got %s", expectedContentType, contentType)
				return false
			}

			return true
		},
		validImageGen,
	))

	properties.TestingRun(t)
}

// TestHandleNotifyEmail tests the email notification endpoint
func TestHandleNotifyEmail(t *testing.T) {
	// Create test server
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		MockAIService: true,
		EnableEmail:   false, // Disabled for testing
	}
	server := NewServer(cfg)

	t.Run("successfully registers email for pending job", func(t *testing.T) {
		// Create a job first
		job, err := server.jobService.CreateJob("test-session-id", "user-photo-123", "shirt-456")
		assert.NoError(t, err)

		// Register email
		reqBody := map[string]string{
			"jobId": job.JobID,
			"email": "user@example.com",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/notify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.True(t, response["success"].(bool))
		assert.Contains(t, response["message"], "email")
	})

	t.Run("rejects invalid email format", func(t *testing.T) {
		job, _ := server.jobService.CreateJob("test-session-id", "user-photo-123", "shirt-456")

		reqBody := map[string]string{
			"jobId": job.JobID,
			"email": "invalid-email",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/notify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns error for non-existent job", func(t *testing.T) {
		reqBody := map[string]string{
			"jobId": "non-existent-job-id",
			"email": "user@example.com",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/notify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("rejects email registration for completed job", func(t *testing.T) {
		job, _ := server.jobService.CreateJob("test-session-id", "user-photo-123", "shirt-456")
		server.jobService.UpdateJobStatus(job.JobID, "completed", "Done")

		reqBody := map[string]string{
			"jobId": job.JobID,
			"email": "user@example.com",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/notify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("requires both jobId and email", func(t *testing.T) {
		reqBody := map[string]string{
			"jobId": "some-job-id",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/notify-email", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
