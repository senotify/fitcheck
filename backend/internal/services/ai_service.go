package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"virtual-fitcheck/internal/models"
)

// AIService handles communication with the generative AI service
type AIService struct {
	client              *http.Client
	serviceURL          string
	apiKey              string
	mockMode            bool
	cloudinaryService   *CloudinaryService
}

// NewAIService creates a new AIService instance
func NewAIService(serviceURL, apiKey string, mockMode bool, cloudinaryService *CloudinaryService) *AIService {
	return &AIService{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		serviceURL:        serviceURL,
		apiKey:            apiKey,
		mockMode:          mockMode,
		cloudinaryService: cloudinaryService,
	}
}

// uploadToCloudinary uploads an image to Cloudinary and returns the public URL
func (s *AIService) uploadToCloudinary(imageData []byte, prefix string) (string, error) {
	if s.cloudinaryService == nil {
		// Fallback to base64 if Cloudinary is not configured
		contentType := detectImageType(imageData)
		b64 := base64.StdEncoding.EncodeToString(imageData)
		return "data:" + contentType + ";base64," + b64, nil
	}
	
	filename := fmt.Sprintf("%s_%d", prefix, time.Now().Unix())
	return s.cloudinaryService.UploadImage(imageData, filename)
}

// detectImageType detects the MIME type of an image from its magic bytes
func detectImageType(imageData []byte) string {
	if len(imageData) < 12 {
		return "image/jpeg" // default
	}

	// Check JPEG
	if imageData[0] == 0xFF && imageData[1] == 0xD8 && imageData[2] == 0xFF {
		return "image/jpeg"
	}

	// Check PNG
	if imageData[0] == 0x89 && imageData[1] == 0x50 && imageData[2] == 0x4E && imageData[3] == 0x47 {
		return "image/png"
	}

	// Check WebP
	if imageData[0] == 0x52 && imageData[1] == 0x49 && imageData[2] == 0x46 && imageData[3] == 0x46 &&
		imageData[8] == 0x57 && imageData[9] == 0x45 && imageData[10] == 0x42 && imageData[11] == 0x50 {
		return "image/webp"
	}

	return "image/jpeg" // default
}

// SubmitTryOnJob submits a virtual try-on job to the AI service
// Implements retry logic with exponential backoff (3 attempts)
// Requirements: 3.2, 8.1, 8.2, 8.4, 8.5
func (s *AIService) SubmitTryOnJob(userPhoto []byte, shirtImage []byte) (string, error) {
	// Mock mode: return a fake job ID immediately
	if s.mockMode {
		return "mock-job-" + fmt.Sprintf("%d", time.Now().Unix()), nil
	}

	// Upload images to Cloudinary to get public URLs (much faster than base64!)
	fmt.Println("Uploading images to Cloudinary...")
	
	personURL, err := s.uploadToCloudinary(userPhoto, "person")
	if err != nil {
		return "", fmt.Errorf("failed to upload person image: %w", err)
	}
	fmt.Printf("Person image uploaded: %s\n", personURL)
	
	garmentURL, err := s.uploadToCloudinary(shirtImage, "garment")
	if err != nil {
		return "", fmt.Errorf("failed to upload garment image: %w", err)
	}
	fmt.Printf("Garment image uploaded: %s\n", garmentURL)

	// Create request payload in Replicate format using URLs
	// Using subhash25rawat/flux-vton model
	request := models.AITryOnRequest{
		Version: "a02643ce418c0e12bad371c4adbfaec0dd1cb34b034ef37650ef205f92ad6199",
		Input: models.ReplicateInput{
			Image:   personURL,
			Garment: garmentURL,
			Part:    "upper_body", // "upper_body" for shirts
		},
	}
	
	// Log the request for debugging
	fmt.Printf("Submitting to Replicate with Cloudinary URLs - Version: %s\n", request.Version)
	fmt.Printf("Person URL: %s\n", personURL)
	fmt.Printf("Garment URL: %s\n", garmentURL)

	// Retry logic with exponential backoff
	var lastErr error
	maxAttempts := 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s, 16s
			backoffDuration := time.Duration(2<<uint(attempt-1)) * time.Second
			fmt.Printf("Retry attempt %d/%d after %v\n", attempt+1, maxAttempts, backoffDuration)
			time.Sleep(backoffDuration)
		}

		jobID, err := s.submitRequest(request)
		if err == nil {
			return jobID, nil
		}
		lastErr = err
		fmt.Printf("Attempt %d failed: %v\n", attempt+1, err)
	}

	// All retry attempts failed
	return "", fmt.Errorf("AI service unavailable after %d attempts: %w", maxAttempts, lastErr)
}

// submitRequest makes a single request to the AI service
func (s *AIService) submitRequest(request models.AITryOnRequest) (string, error) {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log the request body for debugging (first 500 chars)
	bodyStr := string(requestBody)
	if len(bodyStr) > 500 {
		fmt.Printf("Request body (truncated): %s...\n", bodyStr[:500])
	} else {
		fmt.Printf("Request body: %s\n", bodyStr)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", s.serviceURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "wait")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting (429)
	if resp.StatusCode == http.StatusTooManyRequests {
		// Parse the retry_after from the response if available
		var rateLimitResp struct {
			Detail     string `json:"detail"`
			Status     int    `json:"status"`
			RetryAfter int    `json:"retry_after"`
		}
		if err := json.Unmarshal(body, &rateLimitResp); err == nil && rateLimitResp.RetryAfter > 0 {
			// Wait for the specified retry_after duration plus a small buffer
			waitDuration := time.Duration(rateLimitResp.RetryAfter+1) * time.Second
			fmt.Printf("Rate limited. Waiting %v before retry...\n", waitDuration)
			time.Sleep(waitDuration)
		}
		return "", fmt.Errorf("rate limited: %s", string(body))
	}

	// Check status code - Replicate returns 200 (OK), 201 (Created), or 202 (Accepted)
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		return "", fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response models.AITryOnResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return response.ID, nil
}

// CheckJobStatus polls the AI service for job status
// Requirements: 8.3
func (s *AIService) CheckJobStatus(jobID string) (*models.AITryOnResponse, error) {
	// Mock mode: return succeeded status immediately
	if s.mockMode {
		status := "succeeded"
		output := "mock-result-data"
		return &models.AITryOnResponse{
			ID:     jobID,
			Status: status,
			Output: &output,
		}, nil
	}

	// Create HTTP request - use the correct Replicate endpoint
	// The jobID is the prediction ID, so we query: /v1/predictions/{id}
	statusURL := "https://api.replicate.com/v1/predictions/" + jobID
	req, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response models.AITryOnResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetResult retrieves the final image from the AI service
// Requirements: 8.3
func (s *AIService) GetResult(jobID string) ([]byte, error) {
	// Mock mode: return a sample image file
	if s.mockMode {
		// Try to read the sample image file from backend directory
		mockImagePath := "Screenshot 2025-05-04 114255.png"
		mockImage, err := os.ReadFile(mockImagePath)
		if err != nil {
			// Log the error for debugging
			fmt.Printf("Mock mode: Failed to read %s: %v\n", mockImagePath, err)
			
			// Try absolute path
			mockImagePath = "E:\\code\\fitcheck\\backend\\Screenshot 2025-05-04 114255.png"
			mockImage, err = os.ReadFile(mockImagePath)
			if err != nil {
				fmt.Printf("Mock mode: Failed to read absolute path %s: %v\n", mockImagePath, err)
				
				// If file doesn't exist, return a minimal valid PNG image (1x1 transparent pixel)
				mockImage = []byte{
					0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
					0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
					0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
					0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
					0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, // IDAT chunk
					0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
					0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
					0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
					0x42, 0x60, 0x82,
				}
			} else {
				fmt.Printf("Mock mode: Successfully loaded image from absolute path (%d bytes)\n", len(mockImage))
			}
		} else {
			fmt.Printf("Mock mode: Successfully loaded image from relative path (%d bytes)\n", len(mockImage))
		}
		return mockImage, nil
	}

	// First check the job status to get the result URL or data
	status, err := s.CheckJobStatus(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to check job status: %w", err)
	}

	// Check if job is completed
	if status.Status != "succeeded" {
		return nil, fmt.Errorf("job not completed: status is %s", status.Status)
	}

	// Check if output is available
	if status.Output == nil {
		return nil, fmt.Errorf("no output available for job %s", jobID)
	}

	// Output can be a string URL or an array of URLs
	var outputURL string
	switch v := status.Output.(type) {
	case string:
		outputURL = v
	case []interface{}:
		if len(v) > 0 {
			if url, ok := v[0].(string); ok {
				outputURL = url
			}
		}
	default:
		return nil, fmt.Errorf("unexpected output format: %T", status.Output)
	}

	if outputURL == "" {
		return nil, fmt.Errorf("no output URL available for job %s", jobID)
	}

	// Download the result image from the URL
	req, err := http.NewRequest("GET", outputURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	// Add authentication header (some services require it for result download)
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download result: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download result: status %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read result image: %w", err)
	}

	return imageData, nil
}
