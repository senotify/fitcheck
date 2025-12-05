package services

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"virtual-fitcheck/internal/models"
)

// TestAIService_SubmitTryOnJob_Success tests successful job submission
func TestAIService_SubmitTryOnJob_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" || r.URL.Path != "/try-on" {
			t.Errorf("Expected POST /try-on, got %s %s", r.Method, r.URL.Path)
		}

		// Verify authentication header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got '%s'", authHeader)
		}

		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		// Parse request body
		var req models.AITryOnRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify images are base64 encoded
		if _, err := base64.StdEncoding.DecodeString(req.PersonImage); err != nil {
			t.Errorf("PersonImage is not valid base64: %v", err)
		}
		if _, err := base64.StdEncoding.DecodeString(req.GarmentImage); err != nil {
			t.Errorf("GarmentImage is not valid base64: %v", err)
		}

		// Return success response
		response := models.AITryOnResponse{
			ID:     "job-123",
			Status: "starting",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create AI service
	aiService := NewAIService(server.URL, "test-api-key")

	// Test data
	userPhoto := []byte("fake user photo data")
	shirtImage := []byte("fake shirt image data")

	// Submit job
	jobID, err := aiService.SubmitTryOnJob(userPhoto, shirtImage)
	if err != nil {
		t.Fatalf("SubmitTryOnJob failed: %v", err)
	}

	if jobID != "job-123" {
		t.Errorf("Expected job ID 'job-123', got '%s'", jobID)
	}
}

// TestAIService_SubmitTryOnJob_RetryOnFailure tests retry logic
func TestAIService_SubmitTryOnJob_RetryOnFailure(t *testing.T) {
	attemptCount := 0

	// Create mock server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		if attemptCount < 3 {
			// Fail first two attempts
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Succeed on third attempt
		response := models.AITryOnResponse{
			ID:     "job-456",
			Status: "starting",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create AI service
	aiService := NewAIService(server.URL, "test-api-key")

	// Test data
	userPhoto := []byte("fake user photo data")
	shirtImage := []byte("fake shirt image data")

	// Submit job - should succeed after retries
	jobID, err := aiService.SubmitTryOnJob(userPhoto, shirtImage)
	if err != nil {
		t.Fatalf("SubmitTryOnJob failed after retries: %v", err)
	}

	if jobID != "job-456" {
		t.Errorf("Expected job ID 'job-456', got '%s'", jobID)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestAIService_SubmitTryOnJob_AllRetriesFail tests exhausted retries
func TestAIService_SubmitTryOnJob_AllRetriesFail(t *testing.T) {
	attemptCount := 0

	// Create mock server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create AI service
	aiService := NewAIService(server.URL, "test-api-key")

	// Test data
	userPhoto := []byte("fake user photo data")
	shirtImage := []byte("fake shirt image data")

	// Submit job - should fail after all retries
	_, err := aiService.SubmitTryOnJob(userPhoto, shirtImage)
	if err == nil {
		t.Fatal("Expected error after all retries failed, got nil")
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestAIService_CheckJobStatus tests job status checking
func TestAIService_CheckJobStatus(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "GET" || r.URL.Path != "/try-on/job-123" {
			t.Errorf("Expected GET /try-on/job-123, got %s %s", r.Method, r.URL.Path)
		}

		// Verify authentication header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got '%s'", authHeader)
		}

		// Return status response
		response := models.AITryOnResponse{
			ID:     "job-123",
			Status: "processing",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create AI service
	aiService := NewAIService(server.URL, "test-api-key")

	// Check status
	status, err := aiService.CheckJobStatus("job-123")
	if err != nil {
		t.Fatalf("CheckJobStatus failed: %v", err)
	}

	if status.ID != "job-123" {
		t.Errorf("Expected job ID 'job-123', got '%s'", status.ID)
	}

	if status.Status != "processing" {
		t.Errorf("Expected status 'processing', got '%s'", status.Status)
	}
}

// TestAIService_GetResult_Base64 tests getting result as base64
func TestAIService_GetResult_Base64(t *testing.T) {
	testImageData := []byte("fake image data")
	base64Output := base64.StdEncoding.EncodeToString(testImageData)

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return completed job with base64 output
		response := models.AITryOnResponse{
			ID:     "job-123",
			Status: "succeeded",
			Output: &base64Output,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create AI service
	aiService := NewAIService(server.URL, "test-api-key")

	// Get result
	result, err := aiService.GetResult("job-123")
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}

	if string(result) != string(testImageData) {
		t.Errorf("Expected result '%s', got '%s'", testImageData, result)
	}
}

// TestAIService_GetResult_URL tests getting result from URL
func TestAIService_GetResult_URL(t *testing.T) {
	testImageData := []byte("fake image data from url")

	// Create image server
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(testImageData)
	}))
	defer imageServer.Close()

	// Create mock AI service server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return completed job with URL output
		output := imageServer.URL + "/result.jpg"
		response := models.AITryOnResponse{
			ID:     "job-123",
			Status: "succeeded",
			Output: &output,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create AI service
	aiService := NewAIService(server.URL, "test-api-key")

	// Get result
	result, err := aiService.GetResult("job-123")
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}

	if string(result) != string(testImageData) {
		t.Errorf("Expected result '%s', got '%s'", testImageData, result)
	}
}

// TestAIService_GetResult_NotCompleted tests error when job not completed
func TestAIService_GetResult_NotCompleted(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return processing job
		response := models.AITryOnResponse{
			ID:     "job-123",
			Status: "processing",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create AI service
	aiService := NewAIService(server.URL, "test-api-key")

	// Get result - should fail
	_, err := aiService.GetResult("job-123")
	if err == nil {
		t.Fatal("Expected error when job not completed, got nil")
	}
}

// Feature: virtual-fitcheck, Property 21: AI service request formatting
// Validates: Requirements 8.1
// For any processing request sent to the Generative AI Service, the request payload
// should conform to the AI service's API specification with properly formatted image data and required fields.
func TestProperty_AIServiceRequestFormatting(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("AI service requests are properly formatted", prop.ForAll(
		func(userPhotoSize, shirtImageSize int) bool {
			// Generate random image data
			userPhoto := make([]byte, userPhotoSize)
			shirtImage := make([]byte, shirtImageSize)
			for i := range userPhoto {
				userPhoto[i] = byte(i % 256)
			}
			for i := range shirtImage {
				shirtImage[i] = byte((i + 100) % 256)
			}

			// Track if request was properly formatted
			requestValid := false

			// Create mock server to validate request format
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify HTTP method
				if r.Method != "POST" {
					return
				}

				// Verify path
				if r.URL.Path != "/try-on" {
					return
				}

				// Verify Content-Type header
				if r.Header.Get("Content-Type") != "application/json" {
					return
				}

				// Verify Authorization header is present
				if r.Header.Get("Authorization") == "" {
					return
				}

				// Read and parse request body
				body, err := io.ReadAll(r.Body)
				if err != nil {
					return
				}

				var req models.AITryOnRequest
				if err := json.Unmarshal(body, &req); err != nil {
					return
				}

				// Verify required fields are present
				if req.PersonImage == "" || req.GarmentImage == "" {
					return
				}

				// Verify images are valid base64
				decodedPerson, err := base64.StdEncoding.DecodeString(req.PersonImage)
				if err != nil {
					return
				}
				decodedGarment, err := base64.StdEncoding.DecodeString(req.GarmentImage)
				if err != nil {
					return
				}

				// Verify decoded images match original data
				if len(decodedPerson) != len(userPhoto) {
					return
				}
				if len(decodedGarment) != len(shirtImage) {
					return
				}

				// Verify Options field structure (can be nil or properly formatted)
				if req.Options != nil {
					// If options are present, verify they have valid values
					if req.Options.Quality != "" && req.Options.Quality != "standard" && req.Options.Quality != "high" {
						return
					}
				}

				// All validations passed
				requestValid = true

				// Return success response
				response := models.AITryOnResponse{
					ID:     "test-job-id",
					Status: "starting",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create AI service
			aiService := NewAIService(server.URL, "test-api-key")

			// Submit job
			_, err := aiService.SubmitTryOnJob(userPhoto, shirtImage)

			// Request should succeed and be properly formatted
			return err == nil && requestValid
		},
		gen.IntRange(1, 1024*100),  // User photo size: 1 byte to 100KB
		gen.IntRange(1, 1024*100),  // Shirt image size: 1 byte to 100KB
	))

	properties.TestingRun(t)
}

// Feature: virtual-fitcheck, Property 22: AI service authentication
// Validates: Requirements 8.2
// For any request to the Generative AI Service, the HTTP request should include
// valid authentication credentials in the headers.
func TestProperty_AIServiceAuthentication(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("All AI service requests include authentication headers", prop.ForAll(
		func(apiKey string, userPhotoSize, shirtImageSize int) bool {
			// Skip empty API keys as they're not valid test cases
			if apiKey == "" {
				return true
			}

			// Generate random image data
			userPhoto := make([]byte, userPhotoSize)
			shirtImage := make([]byte, shirtImageSize)
			for i := range userPhoto {
				userPhoto[i] = byte(i % 256)
			}
			for i := range shirtImage {
				shirtImage[i] = byte((i + 100) % 256)
			}

			// Track authentication header presence and correctness
			authHeaderCorrect := true
			requestCount := 0

			// Create valid base64 output for GetResult
			testImageData := []byte("test result image")
			base64Output := base64.StdEncoding.EncodeToString(testImageData)

			// Create mock server to validate authentication
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				// Verify Authorization header is present
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					authHeaderCorrect = false
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				// Verify Authorization header format (Bearer token)
				expectedAuth := "Bearer " + apiKey
				if authHeader != expectedAuth {
					authHeaderCorrect = false
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				// Return appropriate response based on endpoint
				if r.Method == "POST" && r.URL.Path == "/try-on" {
					// SubmitTryOnJob endpoint
					response := models.AITryOnResponse{
						ID:     "test-job-id",
						Status: "starting",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(response)
				} else if r.Method == "GET" && r.URL.Path == "/try-on/test-job-id" {
					// CheckJobStatus endpoint - return succeeded with valid base64 output
					response := models.AITryOnResponse{
						ID:     "test-job-id",
						Status: "succeeded",
						Output: &base64Output,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Create AI service with the generated API key
			aiService := NewAIService(server.URL, apiKey)

			// Test SubmitTryOnJob - should include auth header
			jobID, err := aiService.SubmitTryOnJob(userPhoto, shirtImage)
			if err != nil {
				return false
			}

			// Test CheckJobStatus - should include auth header
			_, err = aiService.CheckJobStatus(jobID)
			if err != nil {
				return false
			}

			// Test GetResult - should include auth header
			// Note: GetResult internally calls CheckJobStatus, so this adds 1 more request
			_, err = aiService.GetResult(jobID)
			if err != nil {
				return false
			}

			// Expected: 4 requests total (SubmitTryOnJob, CheckJobStatus, GetResult->CheckJobStatus, GetResult decode)
			// All should have correct authentication
			return requestCount >= 3 && authHeaderCorrect
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 100 }), // API key
		gen.IntRange(1, 1024*10),  // User photo size: 1 byte to 10KB (smaller for faster tests)
		gen.IntRange(1, 1024*10),  // Shirt image size: 1 byte to 10KB
	))

	properties.TestingRun(t)
}

// Feature: virtual-fitcheck, Property 24: Retry with exponential backoff
// Validates: Requirements 8.4
// For any failed request to the Generative AI Service, the system should retry
// up to 3 times with exponentially increasing delays between attempts.
func TestProperty_RetryWithExponentialBackoff(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for number of failures before success (0-3)
	genFailureCount := gen.IntRange(0, 3)

	properties.Property("Failed requests retry with exponential backoff", prop.ForAll(
		func(failureCount int, userPhotoSize, shirtImageSize int) bool {
			// Generate random image data
			userPhoto := make([]byte, userPhotoSize)
			shirtImage := make([]byte, shirtImageSize)
			for i := range userPhoto {
				userPhoto[i] = byte(i % 256)
			}
			for i := range shirtImage {
				shirtImage[i] = byte((i + 100) % 256)
			}

			// Track request attempts and timing
			attemptCount := 0
			var attemptTimes []int64

			// Create mock server that fails N times then succeeds
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attemptCount++
				attemptTimes = append(attemptTimes, time.Now().UnixNano())

				if attemptCount <= failureCount {
					// Fail this attempt
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Service temporarily unavailable"))
					return
				}

				// Succeed on this attempt
				response := models.AITryOnResponse{
					ID:     "test-job-id",
					Status: "starting",
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			// Create AI service
			aiService := NewAIService(server.URL, "test-api-key")

			// Submit job
			startTime := time.Now()
			jobID, err := aiService.SubmitTryOnJob(userPhoto, shirtImage)
			endTime := time.Now()

			// Verify retry behavior based on failure count
			if failureCount < 3 {
				// Should eventually succeed after retries
				if err != nil {
					t.Logf("Expected success after %d failures, got error: %v", failureCount, err)
					return false
				}

				if jobID == "" {
					t.Logf("Expected valid job ID after retries, got empty string")
					return false
				}

				// Verify correct number of attempts (failureCount + 1 for success)
				expectedAttempts := failureCount + 1
				if attemptCount != expectedAttempts {
					t.Logf("Expected %d attempts, got %d", expectedAttempts, attemptCount)
					return false
				}

				// Verify exponential backoff timing
				// Expected delays: 0s (first attempt), 1s (after 1st failure), 2s (after 2nd failure)
				if failureCount > 0 {
					totalDuration := endTime.Sub(startTime)
					
					// Calculate expected minimum duration based on exponential backoff
					// Backoff delays: 1s, 2s, 4s (but we only use what we need)
					var expectedMinDuration time.Duration
					for i := 0; i < failureCount; i++ {
						expectedMinDuration += time.Duration(1<<uint(i)) * time.Second
					}

					// Allow some tolerance for execution time (500ms)
					tolerance := 500 * time.Millisecond
					if totalDuration < (expectedMinDuration - tolerance) {
						t.Logf("Backoff too short: expected at least %v, got %v", expectedMinDuration, totalDuration)
						return false
					}

					// Verify delays between attempts are approximately exponential
					// We check that each delay is at least the expected backoff minus tolerance
					for i := 1; i < len(attemptTimes); i++ {
						delay := time.Duration(attemptTimes[i] - attemptTimes[i-1])
						expectedDelay := time.Duration(1<<uint(i-1)) * time.Second
						
						// Allow 500ms tolerance for execution overhead
						if delay < (expectedDelay - tolerance) {
							t.Logf("Delay %d too short: expected ~%v, got %v", i, expectedDelay, delay)
							return false
						}

						// Also verify delay isn't excessively long (more than 2x expected + tolerance)
						maxDelay := expectedDelay*2 + tolerance
						if delay > maxDelay {
							t.Logf("Delay %d too long: expected ~%v, got %v", i, expectedDelay, delay)
							return false
						}
					}
				}
			} else {
				// failureCount == 3: all retries should fail
				if err == nil {
					t.Logf("Expected error after 3 failures, got success")
					return false
				}

				// Verify error message indicates service unavailability
				if err.Error() == "" {
					t.Logf("Expected non-empty error message")
					return false
				}

				// Verify exactly 3 attempts were made
				if attemptCount != 3 {
					t.Logf("Expected exactly 3 attempts when all fail, got %d", attemptCount)
					return false
				}

				// Verify total duration includes backoff delays (1s + 2s = 3s minimum)
				totalDuration := endTime.Sub(startTime)
				expectedMinDuration := 3 * time.Second // 1s + 2s
				tolerance := 500 * time.Millisecond

				if totalDuration < (expectedMinDuration - tolerance) {
					t.Logf("Total duration too short for 3 attempts: expected at least %v, got %v", expectedMinDuration, totalDuration)
					return false
				}
			}

			return true
		},
		genFailureCount,
		gen.IntRange(1, 1024*10),  // User photo size: 1 byte to 10KB
		gen.IntRange(1, 1024*10),  // Shirt image size: 1 byte to 10KB
	))

	properties.TestingRun(t)
}

// Feature: virtual-fitcheck, Property 25: Exhausted retry notification
// Validates: Requirements 8.5
// For any processing request where all 3 retry attempts to the AI service fail,
// the system should log the error and notify the user that the service is temporarily unavailable.
func TestProperty_ExhaustedRetryNotification(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for different types of failures
	genStatusCode := gen.OneConstOf(
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusGatewayTimeout,
	)

	properties.Property("Exhausted retries return unavailable error message", prop.ForAll(
		func(statusCode int, userPhotoSize, shirtImageSize int) bool {
			// Generate random image data
			userPhoto := make([]byte, userPhotoSize)
			shirtImage := make([]byte, shirtImageSize)
			for i := range userPhoto {
				userPhoto[i] = byte(i % 256)
			}
			for i := range shirtImage {
				shirtImage[i] = byte((i + 100) % 256)
			}

			// Track number of attempts
			attemptCount := 0

			// Create mock server that always fails
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attemptCount++
				w.WriteHeader(statusCode)
				w.Write([]byte("Service error"))
			}))
			defer server.Close()

			// Create AI service
			aiService := NewAIService(server.URL, "test-api-key")

			// Submit job - should fail after all retries
			jobID, err := aiService.SubmitTryOnJob(userPhoto, shirtImage)

			// Verify error is returned
			if err == nil {
				t.Logf("Expected error after all retries failed, got nil")
				return false
			}

			// Verify job ID is empty on failure
			if jobID != "" {
				t.Logf("Expected empty job ID on failure, got %s", jobID)
				return false
			}

			// Verify exactly 3 attempts were made
			if attemptCount != 3 {
				t.Logf("Expected exactly 3 attempts, got %d", attemptCount)
				return false
			}

			// Verify error message indicates service unavailability
			errorMsg := err.Error()
			
			// Error message should contain "AI service unavailable" or similar
			if errorMsg == "" {
				t.Logf("Expected non-empty error message")
				return false
			}

			// Error message should mention "unavailable" and "3 attempts"
			hasUnavailable := false
			hasAttempts := false
			
			// Check for "unavailable" (case-insensitive)
			lowerMsg := errorMsg
			if len(lowerMsg) > 0 {
				for i := 0; i < len(lowerMsg); i++ {
					if lowerMsg[i] >= 'A' && lowerMsg[i] <= 'Z' {
						lowerMsg = lowerMsg[:i] + string(lowerMsg[i]+32) + lowerMsg[i+1:]
					}
				}
			}
			
			if len(lowerMsg) >= 11 {
				for i := 0; i <= len(lowerMsg)-11; i++ {
					if lowerMsg[i:i+11] == "unavailable" {
						hasUnavailable = true
						break
					}
				}
			}

			// Check for "3 attempts" or "3"
			if len(errorMsg) > 0 {
				for i := 0; i < len(errorMsg); i++ {
					if errorMsg[i] == '3' {
						hasAttempts = true
						break
					}
				}
			}

			if !hasUnavailable {
				t.Logf("Error message should mention 'unavailable': %s", errorMsg)
				return false
			}

			if !hasAttempts {
				t.Logf("Error message should mention '3 attempts': %s", errorMsg)
				return false
			}

			return true
		},
		genStatusCode,
		gen.IntRange(1, 1024*10),  // User photo size: 1 byte to 10KB
		gen.IntRange(1, 1024*10),  // Shirt image size: 1 byte to 10KB
	))

	properties.TestingRun(t)
}

// Feature: virtual-fitcheck, Property 23: AI service response parsing
// Validates: Requirements 8.3
// For any valid response from the Generative AI Service, the system should successfully
// parse the response and extract the result URL or image data.
func TestProperty_AIServiceResponseParsing(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for valid status values
	genStatus := gen.OneConstOf("starting", "processing", "succeeded", "failed")

	// Generator for optional output (URL or base64 encoded string)
	genOutput := gen.OneGenOf(
		gen.Const((*string)(nil)), // No output
		gen.PtrOf(gen.Identifier()), // URL-like string
		gen.PtrOf(gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 100 })), // String data
	)

	// Generator for optional error message
	genError := gen.OneGenOf(
		gen.Const((*string)(nil)), // No error
		gen.PtrOf(gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 100 })), // Error message
	)

	properties.Property("AI service responses are successfully parsed", prop.ForAll(
		func(jobID string, status string, output *string, errorMsg *string) bool {
			// Skip invalid test cases
			if jobID == "" {
				return true
			}

			// Create mock server that returns the generated response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Handle both SubmitTryOnJob and CheckJobStatus
				response := models.AITryOnResponse{
					ID:     jobID,
					Status: status,
					Output: output,
					Error:  errorMsg,
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Logf("Failed to encode response: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}))
			defer server.Close()

			// Create AI service
			aiService := NewAIService(server.URL, "test-api-key")

			// Test CheckJobStatus - should successfully parse the response
			parsedResponse, err := aiService.CheckJobStatus(jobID)
			if err != nil {
				// Parsing should not fail for valid JSON responses
				t.Logf("Failed to parse response: %v", err)
				return false
			}

			// Verify all fields were correctly parsed
			if parsedResponse.ID != jobID {
				t.Logf("Job ID mismatch: expected %s, got %s", jobID, parsedResponse.ID)
				return false
			}

			if parsedResponse.Status != status {
				t.Logf("Status mismatch: expected %s, got %s", status, parsedResponse.Status)
				return false
			}

			// Verify output field
			if output == nil && parsedResponse.Output != nil {
				t.Logf("Output should be nil but got %v", *parsedResponse.Output)
				return false
			}
			if output != nil && parsedResponse.Output == nil {
				t.Logf("Output should not be nil but got nil")
				return false
			}
			if output != nil && parsedResponse.Output != nil && *output != *parsedResponse.Output {
				t.Logf("Output mismatch: expected %s, got %s", *output, *parsedResponse.Output)
				return false
			}

			// Verify error field
			if errorMsg == nil && parsedResponse.Error != nil {
				t.Logf("Error should be nil but got %v", *parsedResponse.Error)
				return false
			}
			if errorMsg != nil && parsedResponse.Error == nil {
				t.Logf("Error should not be nil but got nil")
				return false
			}
			if errorMsg != nil && parsedResponse.Error != nil && *errorMsg != *parsedResponse.Error {
				t.Logf("Error mismatch: expected %s, got %s", *errorMsg, *parsedResponse.Error)
				return false
			}

			// Test SubmitTryOnJob response parsing
			userPhoto := []byte("test photo")
			shirtImage := []byte("test shirt")
			
			parsedJobID, err := aiService.SubmitTryOnJob(userPhoto, shirtImage)
			if err != nil {
				// Should successfully parse job submission response
				t.Logf("Failed to parse submit response: %v", err)
				return false
			}

			// Verify job ID was extracted correctly
			if parsedJobID != jobID {
				t.Logf("Submitted job ID mismatch: expected %s, got %s", jobID, parsedJobID)
				return false
			}

			return true
		},
		gen.Identifier(), // Job ID
		genStatus,        // Status
		genOutput,        // Output
		genError,         // Error
	))

	properties.TestingRun(t)
}
