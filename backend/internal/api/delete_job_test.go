package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"virtual-fitcheck/internal/config"

	"github.com/stretchr/testify/assert"
)

// TestDeleteJob_Success verifies successful job deletion
func TestDeleteJob_Success(t *testing.T) {
	// Setup server
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
		DatabaseURL:   "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable",
	}
	
	// Try to create server - if database is not available, skip test
	server := NewServer(cfg)
	if server.jobService == nil {
		t.Skip("Skipping test: database not available")
		return
	}

	// Upload user photo
	userPhotoData := createTestJPEG()
	userPhotoBody := &bytes.Buffer{}
	userPhotoWriter := multipart.NewWriter(userPhotoBody)
	userPhotoPart, err := userPhotoWriter.CreateFormFile("image", "user.jpg")
	assert.NoError(t, err)
	_, err = io.Copy(userPhotoPart, bytes.NewReader(userPhotoData))
	assert.NoError(t, err)
	err = userPhotoWriter.Close()
	assert.NoError(t, err)

	userPhotoReq := httptest.NewRequest("POST", "/api/upload/user-photo", userPhotoBody)
	userPhotoReq.Header.Set("Content-Type", userPhotoWriter.FormDataContentType())

	userPhotoW := httptest.NewRecorder()
	server.router.ServeHTTP(userPhotoW, userPhotoReq)
	
	// Capture session cookie for subsequent requests
	var sessionCookie string
	for _, cookie := range userPhotoW.Result().Cookies() {
		if cookie.Name == "fitcheck_session" {
			sessionCookie = cookie.Value
			break
		}
	}
	assert.Equal(t, 200, userPhotoW.Code)

	var userPhotoResponse map[string]interface{}
	err = json.Unmarshal(userPhotoW.Body.Bytes(), &userPhotoResponse)
	assert.NoError(t, err)
	userPhotoID := userPhotoResponse["fileId"].(string)

	// Upload shirt image
	shirtData := createTestPNG()
	shirtBody := &bytes.Buffer{}
	shirtWriter := multipart.NewWriter(shirtBody)
	shirtPart, err := shirtWriter.CreateFormFile("image", "shirt.png")
	assert.NoError(t, err)
	_, err = io.Copy(shirtPart, bytes.NewReader(shirtData))
	assert.NoError(t, err)
	err = shirtWriter.Close()
	assert.NoError(t, err)

	shirtReq := httptest.NewRequest("POST", "/api/upload/shirt", shirtBody)
	shirtReq.Header.Set("Content-Type", shirtWriter.FormDataContentType())
	shirtReq.AddCookie(&http.Cookie{Name: "fitcheck_session", Value: sessionCookie})

	shirtW := httptest.NewRecorder()
	server.router.ServeHTTP(shirtW, shirtReq)
	assert.Equal(t, 200, shirtW.Code)

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
	processReq.AddCookie(&http.Cookie{Name: "fitcheck_session", Value: sessionCookie})

	processW := httptest.NewRecorder()
	server.router.ServeHTTP(processW, processReq)
	
	if processW.Code != 200 {
		t.Logf("Process request failed with status %d: %s", processW.Code, processW.Body.String())
	}
	assert.Equal(t, 200, processW.Code)

	var processResponse map[string]interface{}
	err = json.Unmarshal(processW.Body.Bytes(), &processResponse)
	assert.NoError(t, err)
	
	if processResponse["jobId"] == nil {
		t.Logf("Process response: %+v", processResponse)
		t.Fatal("jobId is nil in response")
	}
	jobID := processResponse["jobId"].(string)

	// Delete the job
	deleteReq := httptest.NewRequest("DELETE", "/api/jobs/"+jobID, nil)
	deleteReq.AddCookie(&http.Cookie{Name: "fitcheck_session", Value: sessionCookie})
	deleteW := httptest.NewRecorder()
	server.router.ServeHTTP(deleteW, deleteReq)

	// Assert successful deletion
	assert.Equal(t, 200, deleteW.Code)

	var deleteResponse map[string]interface{}
	err = json.Unmarshal(deleteW.Body.Bytes(), &deleteResponse)
	assert.NoError(t, err)
	assert.True(t, deleteResponse["success"].(bool))
	assert.Equal(t, "Job deleted successfully", deleteResponse["message"])

	// Verify job is deleted - trying to get it should fail
	job, err := server.jobService.GetJob(jobID)
	assert.Error(t, err)
	assert.Nil(t, job)

	// Verify files are deleted
	_, err = server.fileService.GetFile(userPhotoID)
	assert.Error(t, err)

	_, err = server.fileService.GetFile(shirtImageID)
	assert.Error(t, err)
}

// TestDeleteJob_NotFound verifies 404 response for non-existent job
func TestDeleteJob_NotFound(t *testing.T) {
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
		DatabaseURL:   "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable",
	}
	
	server := NewServer(cfg)
	if server.jobService == nil {
		t.Skip("Skipping test: database not available")
		return
	}

	// Try to delete non-existent job
	deleteReq := httptest.NewRequest("DELETE", "/api/jobs/non-existent-job-id", nil)
	deleteW := httptest.NewRecorder()
	server.router.ServeHTTP(deleteW, deleteReq)

	// Should return 404
	assert.Equal(t, 404, deleteW.Code)

	var response map[string]interface{}
	err := json.Unmarshal(deleteW.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "JOB_NOT_FOUND", errorObj["code"])
}

// TestDeleteJob_WrongSession verifies 403 response when trying to delete another session's job
func TestDeleteJob_WrongSession(t *testing.T) {
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		AIServiceURL:  "http://mock-ai-service.com",
		AIServiceKey:  "test-key",
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
		DatabaseURL:   "postgresql://postgres:devpassword@localhost:5432/virtualfitcheck?sslmode=disable",
	}
	
	server := NewServer(cfg)
	if server.jobService == nil {
		t.Skip("Skipping test: database not available")
		return
	}

	// Create a job with a different session ID
	job, err := server.jobService.CreateJob("different-session-id", "user-photo-id", "shirt-id")
	assert.NoError(t, err)

	// Try to delete it with the default test session (which will be different)
	deleteReq := httptest.NewRequest("DELETE", "/api/jobs/"+job.JobID, nil)
	deleteW := httptest.NewRecorder()
	server.router.ServeHTTP(deleteW, deleteReq)

	// Should return 403 Forbidden
	assert.Equal(t, 403, deleteW.Code)

	var response map[string]interface{}
	err = json.Unmarshal(deleteW.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "FORBIDDEN", errorObj["code"])

	// Clean up
	_ = server.jobService.DeleteJob(job.JobID)
}
