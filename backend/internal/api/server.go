package api

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
	"virtual-fitcheck/internal/config"
	"virtual-fitcheck/internal/models"
	"virtual-fitcheck/internal/repository"
	"virtual-fitcheck/internal/services"

	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	config                 *config.Config
	router                 *gin.Engine
	db                     *sql.DB
	fileService            *services.FileService
	imageValidationService *services.ImageValidationService
	jobService             *services.JobService
	aiService              *services.AIService
	emailService           *services.EmailService
	cloudinaryService      *services.CloudinaryService
}

// NewServer creates a new API server instance
func NewServer(cfg *config.Config) *Server {
	router := gin.Default()

	// Add CORS middleware
	router.Use(corsMiddleware())

	// Add security headers middleware
	router.Use(securityHeadersMiddleware())

	// Add rate limiting middleware
	router.Use(rateLimitMiddleware())

	// Add session middleware
	router.Use(sessionMiddleware())

	// Initialize database connection
	var db *sql.DB
	var jobRepo repository.JobRepository
	
	if cfg.DatabaseURL != "" {
		dbConfig := repository.DatabaseConfig{
			URL:         cfg.DatabaseURL,
			MaxConns:    cfg.DatabaseMaxConns,
			MaxIdle:     cfg.DatabaseMaxIdle,
			MaxLifetime: 5 * time.Minute,
		}
		
		var err error
		db, err = repository.NewDatabase(dbConfig)
		if err != nil {
			log.Printf("WARNING: Failed to connect to database: %v", err)
			log.Println("Server will continue without database persistence")
		} else {
			log.Println("Database connection established")
			jobRepo = repository.NewPostgresJobRepository(db)
		}
	} else {
		log.Println("WARNING: DATABASE_URL not configured - database persistence disabled")
	}

	// Initialize services
	fileService, err := services.NewFileService(cfg.StoragePath)
	if err != nil {
		panic("Failed to initialize file service: " + err.Error())
	}

	imageValidationService := services.NewImageValidationService(10) // 10MB max
	
	// Initialize job service with repository if available
	var jobService *services.JobService
	if jobRepo != nil {
		jobService = services.NewJobService(jobRepo)
		log.Println("Job service initialized with database persistence")
	} else {
		panic("Database connection required for job service")
	}
	
	// Initialize Cloudinary service if credentials are provided
	var cloudinaryService *services.CloudinaryService
	if cfg.CloudinaryCloudName != "" && cfg.CloudinaryAPIKey != "" && cfg.CloudinaryAPISecret != "" {
		cloudinaryService = services.NewCloudinaryService(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
		log.Println("Cloudinary service initialized - using URLs for faster processing")
	} else {
		log.Println("Cloudinary not configured - using base64 encoding (slower)")
	}
	
	aiService := services.NewAIService(cfg.AIServiceURL, cfg.AIServiceKey, cfg.MockAIService, cloudinaryService)

	// Initialize email service
	emailService := services.NewEmailService(services.EmailServiceConfig{
		SMTPHost:     cfg.SMTPHost,
		SMTPPort:     cfg.SMTPPort,
		SMTPUsername: cfg.SMTPUsername,
		SMTPPassword: cfg.SMTPPassword,
		FromEmail:    cfg.FromEmail,
		BaseURL:      cfg.BaseURL,
		Enabled:      cfg.EnableEmail,
	})
	if cfg.EnableEmail {
		log.Println("Email service enabled")
	} else {
		log.Println("Email service disabled")
	}

	server := &Server{
		config:                 cfg,
		router:                 router,
		db:                     db,
		fileService:            fileService,
		imageValidationService: imageValidationService,
		jobService:             jobService,
		aiService:              aiService,
		emailService:           emailService,
		cloudinaryService:      cloudinaryService,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	api := s.router.Group("/api")
	{
		// Add database health check middleware for all API routes
		api.Use(s.databaseHealthMiddleware())

		// Upload endpoints
		api.POST("/upload/user-photo", s.handleUploadUserPhoto)
		api.POST("/upload/shirt", s.handleUploadShirt)

		// Preview endpoint
		api.GET("/preview/:fileId", s.handlePreview)

		// Processing endpoints
		api.POST("/process", s.handleProcess)
		api.GET("/status/:jobId", s.handleStatus)

		// Result endpoint
		api.GET("/result/:resultId", s.handleResult)

		// Result link endpoint (for email access)
		api.GET("/result-link/:token", s.handleResultLink)

		// Email notification endpoint
		api.POST("/notify-email", s.handleNotifyEmail)

		// Job management endpoints
		api.GET("/jobs", s.handleGetJobs)
		api.DELETE("/jobs/:jobId", s.handleDeleteJob)
	}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	return s.router.Run(":" + s.config.Port)
}

// handleUploadUserPhoto handles user photo uploads
func (s *Server) handleUploadUserPhoto(c *gin.Context) {
	s.handleUpload(c, "user-photo")
}

// handleUploadShirt handles shirt image uploads
func (s *Server) handleUploadShirt(c *gin.Context) {
	s.handleUpload(c, "shirt")
}

// handleUpload is a common handler for both upload endpoints
func (s *Server) handleUpload(c *gin.Context, uploadType string) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "No image file provided in request",
				Details: err.Error(),
			},
		})
		return
	}
	defer file.Close()

	// Read file contents
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "FILE_READ_ERROR",
				Message: "Failed to read uploaded file",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate file size
	isValidSize, err := s.imageValidationService.ValidateSize(fileBytes)
	if !isValidSize {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "FILE_TOO_LARGE",
				Message: "File size exceeds maximum allowed size of 10MB",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate file format
	isValidFormat, err := s.imageValidationService.ValidateFormat(fileBytes)
	if !isValidFormat {
		c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_FORMAT",
				Message: "Invalid image format. Only JPEG, PNG, and WebP are accepted",
				Details: err.Error(),
			},
		})
		return
	}

	// For user photos, validate human features
	if uploadType == "user-photo" {
		hasHumanFeatures, err := s.imageValidationService.DetectHumanFeatures(fileBytes)
		if !hasHumanFeatures {
			c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "NO_HUMAN_FEATURES",
					Message: "Image must contain detectable human features",
					Details: err.Error(),
				},
			})
			return
		}
	}

	// Upload to Cloudinary if configured, otherwise save locally
	var fileId string
	var previewUrl string
	
	if s.cloudinaryService != nil {
		// Upload to Cloudinary
		cloudinaryURL, err := s.cloudinaryService.UploadImage(fileBytes, header.Filename)
		if err != nil {
			log.Printf("WARNING: Failed to upload to Cloudinary: %v. Falling back to local storage.", err)
			// Fallback to local storage
			fileId = s.fileService.GenerateFileID()
			_, err = s.fileService.SaveUpload(fileBytes, fileId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Success: false,
					Error: models.ErrorDetail{
						Code:    "STORAGE_ERROR",
						Message: "Failed to save uploaded file",
						Details: err.Error(),
					},
				})
				return
			}
			previewUrl = "/api/preview/" + fileId
			
			// Schedule cleanup after 1 hour
			cleanupDelay := time.Duration(s.config.CleanupDelay) * time.Minute
			s.fileService.ScheduleCleanup(fileId, cleanupDelay)
		} else {
			// Use Cloudinary URL as fileId
			fileId = cloudinaryURL
			previewUrl = cloudinaryURL
		}
	} else {
		// No Cloudinary configured, use local storage
		fileId = s.fileService.GenerateFileID()
		_, err = s.fileService.SaveUpload(fileBytes, fileId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "STORAGE_ERROR",
					Message: "Failed to save uploaded file",
					Details: err.Error(),
				},
			})
			return
		}
		previewUrl = "/api/preview/" + fileId
		
		// Schedule cleanup after 1 hour
		cleanupDelay := time.Duration(s.config.CleanupDelay) * time.Minute
		s.fileService.ScheduleCleanup(fileId, cleanupDelay)
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"fileId":     fileId,
		"previewUrl": previewUrl,
		"filename":   header.Filename,
	})
}

// handlePreview serves uploaded image files for preview
// Note: With Cloudinary migration, this endpoint is mainly for backward compatibility
func (s *Server) handlePreview(c *gin.Context) {
	fileId := c.Param("fileId")

	// Try to retrieve file from local storage
	fileBytes, err := s.fileService.GetFile(fileId)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "FILE_NOT_FOUND",
				Message: "The requested file was not found",
				Details: err.Error(),
			},
		})
		return
	}

	// Detect MIME type based on file content
	contentType := detectContentType(fileBytes)

	// Set headers for proper image display
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Access-Control-Allow-Origin", "*")
	
	// Serve the image
	c.Data(http.StatusOK, contentType, fileBytes)
}

// isCloudinaryURL checks if a string is a Cloudinary URL
func isCloudinaryURL(url string) bool {
	return len(url) > 8 && (url[:7] == "http://" || url[:8] == "https://")
}

// detectContentType determines the MIME type based on file magic numbers
func detectContentType(file []byte) string {
	if len(file) < 12 {
		return "application/octet-stream"
	}

	// Check JPEG
	if file[0] == 0xFF && file[1] == 0xD8 && file[2] == 0xFF {
		return "image/jpeg"
	}

	// Check PNG
	if file[0] == 0x89 && file[1] == 0x50 && file[2] == 0x4E && file[3] == 0x47 {
		return "image/png"
	}

	// Check WebP (RIFF container with WEBP signature)
	if file[0] == 0x52 && file[1] == 0x49 && file[2] == 0x46 && file[3] == 0x46 &&
		file[8] == 0x57 && file[9] == 0x45 && file[10] == 0x42 && file[11] == 0x50 {
		return "image/webp"
	}

	return "application/octet-stream"
}

// ProcessRequest represents the request body for the process endpoint
type ProcessRequest struct {
	UserPhotoID  string `json:"userPhotoId" binding:"required"`
	ShirtImageID string `json:"shirtImageId" binding:"required"`
}

// handleProcess initiates virtual try-on processing
// Requirements: 3.1, 3.2, 7.3
func (s *Server) handleProcess(c *gin.Context) {
	var req ProcessRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body. Both userPhotoId and shirtImageId are required",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate that user photo exists (skip validation for Cloudinary URLs)
	if !isCloudinaryURL(req.UserPhotoID) {
		_, err := s.fileService.GetFile(req.UserPhotoID)
		if err != nil {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "USER_PHOTO_NOT_FOUND",
					Message: "User photo not found. Please upload a user photo first",
					Details: err.Error(),
				},
			})
			return
		}
	}

	// Validate that shirt image exists (skip validation for Cloudinary URLs)
	if !isCloudinaryURL(req.ShirtImageID) {
		_, err := s.fileService.GetFile(req.ShirtImageID)
		if err != nil {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "SHIRT_IMAGE_NOT_FOUND",
					Message: "Shirt image not found. Please upload a shirt image first",
					Details: err.Error(),
				},
			})
			return
		}
	}

	// Get session ID from context
	sessionID := GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "SESSION_ERROR",
				Message: "Failed to retrieve session",
			},
		})
		return
	}

	// Create processing job with session ID
	job, err := s.jobService.CreateJob(sessionID, req.UserPhotoID, req.ShirtImageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "JOB_CREATION_FAILED",
				Message: "Failed to create processing job",
				Details: err.Error(),
			},
		})
		return
	}

	// Initiate AI service call asynchronously
	go s.processJobAsync(job.JobID, req.UserPhotoID, req.ShirtImageID)

	// Return response with jobId and estimated time
	// Estimated time is 15 seconds based on typical AI processing times
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"jobId":         job.JobID,
		"estimatedTime": 15, // seconds
	})
}

// sendJobCompletionEmail sends an email notification when a job completes or fails
func (s *Server) sendJobCompletionEmail(jobID string, resultID *string, errorMsg *string) {
	// Get job details
	job, err := s.jobService.GetJob(jobID)
	if err != nil {
		log.Printf("Job %s: Failed to get job for email notification: %v", jobID, err)
		return
	}

	// Check if job has associated email
	if job.Email == nil || *job.Email == "" {
		return
	}

	log.Printf("Job %s: Sending completion email to %s", jobID, *job.Email)

	// Build result URL
	var resultURL string
	if resultID != nil {
		// Job succeeded - build result URL
		if job.ResultToken != nil && *job.ResultToken != "" {
			resultURL = s.config.BaseURL + "/api/result-link/" + *job.ResultToken
		} else {
			resultURL = s.config.BaseURL + "/api/result/" + *resultID
		}
	} else if errorMsg != nil {
		// Job failed - could send a different email or include error info
		// For now, we'll just log it and not send an email for failures
		log.Printf("Job %s: Job failed with error: %s. Not sending email for failed jobs.", jobID, *errorMsg)
		return
	}

	// Send email asynchronously to avoid blocking
	go func(email, url, jid string) {
		err := s.emailService.SendCompletionEmail(email, url, jid)
		if err != nil {
			log.Printf("Job %s: Failed to send completion email: %v", jid, err)
		} else {
			log.Printf("Job %s: Completion email sent successfully to %s", jid, email)
		}
	}(*job.Email, resultURL, jobID)
}

// processJobAsync handles the asynchronous processing of a virtual try-on job
func (s *Server) processJobAsync(jobID, userPhotoID, shirtImageID string) {
	// Update job status to processing
	err := s.jobService.UpdateJobStatus(jobID, models.JobStatusProcessing, "Starting AI processing")
	if err != nil {
		// Log error but continue
		return
	}

	// Retrieve user photo (from Cloudinary URL or local storage)
	var userPhoto []byte
	if isCloudinaryURL(userPhotoID) {
		// For Cloudinary URLs, we don't need to download - AI service can use URL directly
		// Just pass the URL as bytes for now (AI service will handle it)
		userPhoto = []byte(userPhotoID)
	} else {
		var err error
		userPhoto, err = s.fileService.GetFile(userPhotoID)
		if err != nil {
			errorMsg := "Failed to retrieve user photo: " + err.Error()
			s.jobService.SetJobError(jobID, errorMsg)
			s.sendJobCompletionEmail(jobID, nil, &errorMsg)
			return
		}
	}

	// Retrieve shirt image (from Cloudinary URL or local storage)
	var shirtImage []byte
	if isCloudinaryURL(shirtImageID) {
		// For Cloudinary URLs, we don't need to download - AI service can use URL directly
		shirtImage = []byte(shirtImageID)
	} else {
		var err error
		shirtImage, err = s.fileService.GetFile(shirtImageID)
		if err != nil {
			errorMsg := "Failed to retrieve shirt image: " + err.Error()
			s.jobService.SetJobError(jobID, errorMsg)
			s.sendJobCompletionEmail(jobID, nil, &errorMsg)
			return
		}
	}

	// Update progress
	s.jobService.UpdateJobProgress(jobID, 25, "Sending images to AI service")

	// Submit job to AI service
	aiJobID, err := s.aiService.SubmitTryOnJob(userPhoto, shirtImage)
	if err != nil {
		errorMsg := "AI service error: " + err.Error()
		s.jobService.SetJobError(jobID, errorMsg)
		s.sendJobCompletionEmail(jobID, nil, &errorMsg)
		return
	}

	// Update progress
	s.jobService.UpdateJobProgress(jobID, 50, "AI processing in progress")

	// Poll for completion (simplified polling logic)
	// In a production system, this would be more sophisticated
	maxAttempts := 120 // 120 attempts with 5 second intervals = 10 minutes max
	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(5 * time.Second)

		// Check AI job status
		aiStatus, err := s.aiService.CheckJobStatus(aiJobID)
		if err != nil {
			// Continue polling on transient errors
			continue
		}

		// Update progress based on AI status
		if aiStatus.Status == "processing" {
			progress := 50 + (attempt * 40 / maxAttempts) // Progress from 50% to 90%
			s.jobService.UpdateJobProgress(jobID, progress, "AI processing: "+aiStatus.Status)
			continue
		}

		// Check if completed
		if aiStatus.Status == "succeeded" {
			log.Printf("Job %s: AI processing succeeded, retrieving result", jobID)
			
			// Retrieve result
			resultImage, err := s.aiService.GetResult(aiJobID)
			if err != nil {
				log.Printf("Job %s: Failed to retrieve result: %v", jobID, err)
				errorMsg := "Failed to retrieve result: " + err.Error()
				s.jobService.SetJobError(jobID, errorMsg)
				s.sendJobCompletionEmail(jobID, nil, &errorMsg)
				return
			}

			log.Printf("Job %s: Retrieved result image (%d bytes)", jobID, len(resultImage))

			// Save result image
			resultID := s.fileService.GenerateFileID()
			_, err = s.fileService.SaveUpload(resultImage, resultID)
			if err != nil {
				log.Printf("Job %s: Failed to save result: %v", jobID, err)
				errorMsg := "Failed to save result: " + err.Error()
				s.jobService.SetJobError(jobID, errorMsg)
				s.sendJobCompletionEmail(jobID, nil, &errorMsg)
				return
			}

			log.Printf("Job %s: Saved result with ID %s", jobID, resultID)

			// Update job with result
			s.jobService.SetJobResult(jobID, resultID)
			s.jobService.UpdateJobStatus(jobID, models.JobStatusCompleted, "Processing completed successfully")
			s.jobService.UpdateJobProgress(jobID, 100, "Completed")

			log.Printf("Job %s: Completed successfully, result URL: /api/result/%s", jobID, resultID)

			// Send completion email if user registered for notifications
			s.sendJobCompletionEmail(jobID, &resultID, nil)

			// Schedule cleanup for result after 1 hour
			cleanupDelay := time.Duration(s.config.CleanupDelay) * time.Minute
			s.fileService.ScheduleCleanup(resultID, cleanupDelay)
			return
		}

		// Check if failed
		if aiStatus.Status == "failed" {
			errorMsg := "AI processing failed"
			if aiStatus.Error != nil {
				errorMsg = *aiStatus.Error
			}
			s.jobService.SetJobError(jobID, errorMsg)
			s.sendJobCompletionEmail(jobID, nil, &errorMsg)
			return
		}
	}

	// Timeout
	errorMsg := "Processing timeout: exceeded maximum processing time"
	s.jobService.SetJobError(jobID, errorMsg)
	s.sendJobCompletionEmail(jobID, nil, &errorMsg)
}

// handleStatus returns the current status of a processing job
// Requirements: 3.3, 3.4, 3.5, 7.4
func (s *Server) handleStatus(c *gin.Context) {
	jobID := c.Param("jobId")

	// Retrieve job from job service
	job, err := s.jobService.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "JOB_NOT_FOUND",
				Message: "The requested job was not found",
				Details: err.Error(),
			},
		})
		return
	}

	// Build response
	response := gin.H{
		"status":  string(job.Status),
		"progress": job.Progress,
		"message":  job.StatusMessage,
	}

	// Include resultUrl when job is completed
	if job.Status == models.JobStatusCompleted && job.ResultID != nil {
		response["resultUrl"] = "/api/result/" + *job.ResultID
	}

	// Include error message if job failed
	if job.Status == models.JobStatusFailed && job.Error != nil {
		response["error"] = *job.Error
	}

	c.JSON(http.StatusOK, response)
}

// handleResult serves the composite result image for download
// Requirements: 4.1, 4.2, 4.3
func (s *Server) handleResult(c *gin.Context) {
	resultID := c.Param("resultId")

	// If resultID is a Cloudinary URL, redirect to it
	if isCloudinaryURL(resultID) {
		c.Redirect(http.StatusMovedPermanently, resultID)
		return
	}

	// Otherwise, retrieve from local storage (for backward compatibility)
	fileBytes, err := s.fileService.GetFile(resultID)
	if err != nil {
		log.Printf("ERROR: Failed to get result file %s: %v", resultID, err)
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "RESULT_NOT_FOUND",
				Message: "The requested result was not found",
				Details: err.Error(),
			},
		})
		return
	}

	log.Printf("Serving result %s: %d bytes", resultID, len(fileBytes))

	// Detect MIME type based on file content
	contentType := detectContentType(fileBytes)
	log.Printf("Detected content type: %s", contentType)

	// Generate descriptive filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := "virtual-fitcheck-" + timestamp + getFileExtension(contentType)

	// Set Content-Disposition header for download with descriptive filename
	// Requirements: 4.2 - initiate file download with descriptive filename
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	
	// Set Content-Type header
	c.Header("Content-Type", contentType)
	
	// Serve the image without additional compression to preserve quality
	c.Data(http.StatusOK, contentType, fileBytes)
}

// getFileExtension returns the appropriate file extension for a given MIME type
func getFileExtension(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

// handleGetJobs handles job list requests
// Requirements: 1.1, 1.2, 4.1, 4.2
func (s *Server) handleGetJobs(c *gin.Context) {
	// Get session ID from context
	sessionID := GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Session ID is required",
			},
		})
		return
	}

	// Get query parameters
	statusFilter := c.Query("status")
	sortOrder := c.DefaultQuery("sort", "newest")
	limit := 50
	offset := 0

	// Parse limit and offset if provided
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Build filter
	filter := repository.JobFilter{
		SessionID: sessionID,
		SortOrder: sortOrder,
		Limit:     limit,
		Offset:    offset,
	}

	// Add status filter if provided
	if statusFilter != "" {
		filter.Status = &statusFilter
	}

	// Get jobs from repository
	result, err := s.jobService.GetJobsWithFilter(context.Background(), filter)
	if err != nil {
		log.Printf("ERROR: Failed to get jobs for session %s: %v", sessionID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "QUERY_FAILED",
				Message: "Failed to retrieve jobs",
			},
		})
		return
	}

	// Build response with job data
	type JobResponse struct {
		JobID         string  `json:"jobId"`
		Status        string  `json:"status"`
		Progress      int     `json:"progress"`
		StatusMessage string  `json:"statusMessage"`
		ResultURL     *string `json:"resultUrl,omitempty"`
		ThumbnailURL  string  `json:"thumbnailUrl"`
		UserPhotoURL  string  `json:"userPhotoUrl"`
		ShirtImageURL string  `json:"shirtImageUrl"`
		CreatedAt     string  `json:"createdAt"`
		CompletedAt   *string `json:"completedAt,omitempty"`
	}

	jobs := make([]JobResponse, 0, len(result.Jobs))
	for _, job := range result.Jobs {
		// Helper function to build image URL
		buildImageURL := func(fileID string) string {
			// If fileID is already a full URL (Cloudinary), return as-is
			if len(fileID) > 8 && (fileID[:7] == "http://" || fileID[:8] == "https://") {
				return fileID
			}
			// Otherwise, build local preview URL
			return "http://localhost:8080/api/preview/" + fileID
		}
		
		jobResp := JobResponse{
			JobID:         job.JobID,
			Status:        string(job.Status),
			Progress:      job.Progress,
			StatusMessage: job.StatusMessage,
			ThumbnailURL:  buildImageURL(job.UserPhotoID),
			UserPhotoURL:  buildImageURL(job.UserPhotoID),
			ShirtImageURL: buildImageURL(job.ShirtImageID),
			CreatedAt:     job.CreatedAt.Format(time.RFC3339),
		}

		// Add result URL if job is completed
		if job.ResultID != nil {
			resultURL := buildImageURL(*job.ResultID)
			jobResp.ResultURL = &resultURL
		}

		// Add completion time if available
		if job.CompletedAt != nil {
			completedAt := job.CompletedAt.Format(time.RFC3339)
			jobResp.CompletedAt = &completedAt
		}

		jobs = append(jobs, jobResp)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"jobs":    jobs,
		"total":   result.Total,
		"limit":   result.Limit,
		"offset":  result.Offset,
	})
}

// handleDeleteJob handles job deletion requests
// Requirements: 5.2, 5.4
func (s *Server) handleDeleteJob(c *gin.Context) {
	jobID := c.Param("jobId")

	if jobID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Job ID is required",
			},
		})
		return
	}

	// Get session ID from context
	sessionID := GetSessionID(c)
	if sessionID == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "SESSION_ERROR",
				Message: "Failed to retrieve session",
			},
		})
		return
	}

	// Retrieve the job to verify ownership and get file IDs
	job, err := s.jobService.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "JOB_NOT_FOUND",
				Message: "The requested job was not found",
				Details: err.Error(),
			},
		})
		return
	}

	// Verify job belongs to current session
	if job.SessionID != sessionID {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "FORBIDDEN",
				Message: "You do not have permission to delete this job",
			},
		})
		return
	}

	// Delete associated files (only delete local files, not Cloudinary URLs)
	// Delete user photo
	if !isCloudinaryURL(job.UserPhotoID) {
		if err := s.fileService.DeleteFile(job.UserPhotoID); err != nil {
			log.Printf("Warning: Failed to delete user photo %s: %v", job.UserPhotoID, err)
		}
	}

	// Delete shirt image
	if !isCloudinaryURL(job.ShirtImageID) {
		if err := s.fileService.DeleteFile(job.ShirtImageID); err != nil {
			log.Printf("Warning: Failed to delete shirt image %s: %v", job.ShirtImageID, err)
		}
	}

	// Delete result if it exists
	if job.ResultID != nil && !isCloudinaryURL(*job.ResultID) {
		if err := s.fileService.DeleteFile(*job.ResultID); err != nil {
			log.Printf("Warning: Failed to delete result %s: %v", *job.ResultID, err)
		}
	}

	// Delete job from database
	if err := s.jobService.DeleteJob(jobID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "DELETE_FAILED",
				Message: "Failed to delete job from database",
				Details: err.Error(),
			},
		})
		return
	}

	log.Printf("Job %s deleted successfully by session %s", jobID, sessionID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job deleted successfully",
	})
}

// corsMiddleware configures CORS (Cross-Origin Resource Sharing) headers
// Requirements: 6.3
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow requests from frontend origin
		// In production, this should be configured to specific allowed origins
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// securityHeadersMiddleware adds security headers to all responses
// Requirements: 6.4
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Enforce HTTPS in production
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy - relaxed for development
		// In production, tighten this to specific domains
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; img-src * data: blob:; style-src 'self' 'unsafe-inline'")

		// Prevent MIME type sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Writer.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// XSS Protection
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer Policy
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// rateLimitMiddleware implements basic rate limiting
// Requirements: 6.3
func rateLimitMiddleware() gin.HandlerFunc {
	// Simple in-memory rate limiter
	// In production, use a more sophisticated solution like Redis-based rate limiting
	type clientInfo struct {
		requests  int
		resetTime time.Time
	}

	clients := make(map[string]*clientInfo)
	maxRequests := 100 // 100 requests per minute per IP
	window := time.Minute

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Get or create client info
		client, exists := clients[clientIP]
		if !exists || time.Now().After(client.resetTime) {
			clients[clientIP] = &clientInfo{
				requests:  1,
				resetTime: time.Now().Add(window),
			}
			c.Next()
			return
		}

		// Check rate limit
		if client.requests >= maxRequests {
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "RATE_LIMIT_EXCEEDED",
					Message: "Too many requests. Please try again later",
					Details: "Rate limit: " + string(rune(maxRequests)) + " requests per minute",
				},
			})
			c.Abort()
			return
		}

		// Increment request count
		client.requests++
		c.Next()
	}
}

// databaseHealthMiddleware checks database connectivity before processing requests
// Requirements: 3.4
func (s *Server) databaseHealthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip health check for non-database endpoints
		path := c.Request.URL.Path
		if path == "/api/preview/" || path == "/health" {
			c.Next()
			return
		}

		// Check if database is available
		if s.db == nil {
			log.Printf("ERROR: Database connection is not available")
			c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "DATABASE_UNAVAILABLE",
					Message: "The database service is currently unavailable. Please try again later.",
					Details: "Database connection not initialized",
				},
			})
			c.Abort()
			return
		}

		// Perform health check
		if err := repository.CheckDatabaseHealth(s.db); err != nil {
			log.Printf("ERROR: Database health check failed: %v", err)
			c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "DATABASE_UNAVAILABLE",
					Message: "The database service is currently unavailable. Please try again later.",
					Details: "Unable to connect to database",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// handleNotifyEmail handles email notification registration for long-running jobs
// Requirements: 9.2, 9.3
func (s *Server) handleNotifyEmail(c *gin.Context) {
	var req struct {
		JobID string `json:"jobId" binding:"required"`
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate email format
	valid, err := s.emailService.ValidateEmail(req.Email)
	if !valid || err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_EMAIL",
				Message: "Invalid email address format",
				Details: err.Error(),
			},
		})
		return
	}

	// Check if job exists
	job, err := s.jobService.GetJob(req.JobID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "JOB_NOT_FOUND",
				Message: "Job not found",
				Details: req.JobID,
			},
		})
		return
	}

	// Don't allow email registration for already completed jobs
	if job.Status == models.JobStatusCompleted || job.Status == models.JobStatusFailed {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "JOB_ALREADY_COMPLETED",
				Message: "Cannot register email for completed or failed job",
				Details: string(job.Status),
			},
		})
		return
	}

	// Generate a secure token for result access
	token, err := s.emailService.GenerateResultToken()
	if err != nil {
		log.Printf("Failed to generate result token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "TOKEN_GENERATION_FAILED",
				Message: "Failed to generate secure token",
			},
		})
		return
	}

	// Associate email and token with the job
	if err := s.jobService.SetJobEmail(req.JobID, req.Email); err != nil {
		log.Printf("Failed to set job email: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "EMAIL_REGISTRATION_FAILED",
				Message: "Failed to register email for job",
			},
		})
		return
	}

	if err := s.jobService.SetJobResultToken(req.JobID, token); err != nil {
		log.Printf("Failed to set job result token: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "TOKEN_REGISTRATION_FAILED",
				Message: "Failed to register result token",
			},
		})
		return
	}

	log.Printf("Email notification registered for job %s: %s", req.JobID, req.Email)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "You will receive an email when processing completes",
	})
}

// handleResultLink handles result access via email link with token validation
// Requirements: 9.4
func (s *Server) handleResultLink(c *gin.Context) {
	token := c.Param("token")

	if token == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_TOKEN",
				Message: "Token is required",
			},
		})
		return
	}

	// Find job by token
	job, err := s.jobService.GetJobByToken(token)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_TOKEN",
				Message: "Invalid or expired link",
				Details: "The link you followed is invalid or has expired. Links are valid for 24 hours after job completion.",
			},
		})
		return
	}

	// Check if job is completed
	if job.Status != models.JobStatusCompleted {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "JOB_NOT_COMPLETED",
				Message: "Job is not yet completed",
				Details: fmt.Sprintf("Current status: %s", job.Status),
			},
		})
		return
	}

	// Check if job has a result
	if job.ResultID == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "RESULT_NOT_FOUND",
				Message: "Result not found for this job",
			},
		})
		return
	}

	// Check token expiration (24 hours from completion)
	if job.CompletedAt != nil {
		expirationTime := job.CompletedAt.Add(24 * time.Hour)
		if time.Now().After(expirationTime) {
			c.JSON(http.StatusGone, models.ErrorResponse{
				Success: false,
				Error: models.ErrorDetail{
					Code:    "LINK_EXPIRED",
					Message: "This link has expired",
					Details: fmt.Sprintf("Links expire 24 hours after job completion. This link expired at %s", expirationTime.Format(time.RFC1123)),
				},
			})
			return
		}
	}

	// If result is a Cloudinary URL, redirect to it
	if isCloudinaryURL(*job.ResultID) {
		c.Redirect(http.StatusMovedPermanently, *job.ResultID)
		return
	}

	// Otherwise, retrieve from local storage (for backward compatibility)
	fileBytes, err := s.fileService.GetFile(*job.ResultID)
	if err != nil {
		log.Printf("ERROR: Failed to get result file %s: %v", *job.ResultID, err)
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "RESULT_NOT_FOUND",
				Message: "The result file could not be found",
				Details: "The result may have been cleaned up. Please contact support if you need assistance.",
			},
		})
		return
	}

	log.Printf("Serving result via token for job %s: %d bytes", job.JobID, len(fileBytes))

	// Detect MIME type based on file content
	contentType := detectContentType(fileBytes)

	// Generate descriptive filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := "virtual-fitcheck-" + timestamp + getFileExtension(contentType)

	// Set Content-Disposition header for inline display
	c.Header("Content-Disposition", "inline; filename=\""+filename+"\"")
	
	// Set Content-Type header
	c.Header("Content-Type", contentType)
	
	// Serve the image without additional compression to preserve quality
	c.Data(http.StatusOK, contentType, fileBytes)
}
