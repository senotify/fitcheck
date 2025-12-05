package api

import (
	"io"
	"log"
	"net/http"
	"time"
	"virtual-fitcheck/internal/config"
	"virtual-fitcheck/internal/models"
	"virtual-fitcheck/internal/services"

	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	config                 *config.Config
	router                 *gin.Engine
	fileService            *services.FileService
	imageValidationService *services.ImageValidationService
	jobService             *services.JobService
	aiService              *services.AIService
	emailService           *services.EmailService
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

	// Initialize services
	fileService, err := services.NewFileService(cfg.StoragePath)
	if err != nil {
		panic("Failed to initialize file service: " + err.Error())
	}

	imageValidationService := services.NewImageValidationService(10) // 10MB max
	jobService := services.NewJobService()
	
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
		fileService:            fileService,
		imageValidationService: imageValidationService,
		jobService:             jobService,
		aiService:              aiService,
		emailService:           emailService,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	api := s.router.Group("/api")
	{
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

	// Generate unique file ID
	fileId := s.fileService.GenerateFileID()

	// Save file
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

	// Schedule cleanup after 1 hour
	cleanupDelay := time.Duration(s.config.CleanupDelay) * time.Minute
	s.fileService.ScheduleCleanup(fileId, cleanupDelay)

	// Return success response
	previewUrl := "/api/preview/" + fileId
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"fileId":     fileId,
		"previewUrl": previewUrl,
		"filename":   header.Filename,
	})
}

// handlePreview serves uploaded image files for preview
func (s *Server) handlePreview(c *gin.Context) {
	fileId := c.Param("fileId")

	// Retrieve file from storage
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

	// Validate that user photo exists
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

	// Validate that shirt image exists
	_, err = s.fileService.GetFile(req.ShirtImageID)
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

	// Create processing job
	job, err := s.jobService.CreateJob(req.UserPhotoID, req.ShirtImageID)
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

// processJobAsync handles the asynchronous processing of a virtual try-on job
func (s *Server) processJobAsync(jobID, userPhotoID, shirtImageID string) {
	// Update job status to processing
	err := s.jobService.UpdateJobStatus(jobID, models.JobStatusProcessing, "Starting AI processing")
	if err != nil {
		// Log error but continue
		return
	}

	// Retrieve user photo
	userPhoto, err := s.fileService.GetFile(userPhotoID)
	if err != nil {
		s.jobService.SetJobError(jobID, "Failed to retrieve user photo: "+err.Error())
		return
	}

	// Retrieve shirt image
	shirtImage, err := s.fileService.GetFile(shirtImageID)
	if err != nil {
		s.jobService.SetJobError(jobID, "Failed to retrieve shirt image: "+err.Error())
		return
	}

	// Update progress
	s.jobService.UpdateJobProgress(jobID, 25, "Sending images to AI service")

	// Submit job to AI service
	aiJobID, err := s.aiService.SubmitTryOnJob(userPhoto, shirtImage)
	if err != nil {
		s.jobService.SetJobError(jobID, "AI service error: "+err.Error())
		return
	}

	// Update progress
	s.jobService.UpdateJobProgress(jobID, 50, "AI processing in progress")

	// Poll for completion (simplified polling logic)
	// In a production system, this would be more sophisticated
	maxAttempts := 60 // 60 attempts with 5 second intervals = 5 minutes max
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
				s.jobService.SetJobError(jobID, "Failed to retrieve result: "+err.Error())
				return
			}

			log.Printf("Job %s: Retrieved result image (%d bytes)", jobID, len(resultImage))

			// Save result image
			resultID := s.fileService.GenerateFileID()
			_, err = s.fileService.SaveUpload(resultImage, resultID)
			if err != nil {
				log.Printf("Job %s: Failed to save result: %v", jobID, err)
				s.jobService.SetJobError(jobID, "Failed to save result: "+err.Error())
				return
			}

			log.Printf("Job %s: Saved result with ID %s", jobID, resultID)

			// Update job with result
			s.jobService.SetJobResult(jobID, resultID)
			s.jobService.UpdateJobStatus(jobID, models.JobStatusCompleted, "Processing completed successfully")
			s.jobService.UpdateJobProgress(jobID, 100, "Completed")

			log.Printf("Job %s: Completed successfully, result URL: /api/result/%s", jobID, resultID)

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
			return
		}
	}

	// Timeout
	s.jobService.SetJobError(jobID, "Processing timeout: exceeded maximum processing time")
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

	// Retrieve result file from storage
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

	// Set Content-Disposition header for inline display (not download)
	c.Header("Content-Disposition", "inline; filename=\""+filename+"\"")
	
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
