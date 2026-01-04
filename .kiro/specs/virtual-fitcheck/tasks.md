# Implementation Plan

- [x] 1. Set up project structure and initialize repositories

  - Create frontend directory with React + TypeScript setup
  - Create backend directory with Go module initialization
  - Set up basic folder structure for both frontend and backend
  - Initialize git repository with .gitignore files
  - _Requirements: All_

- [x] 2. Implement backend data models and types

  - Create Go structs for UploadedFile, ProcessingJob, AITryOnRequest, AITryOnResponse
  - Define JobStatus constants and error response types
  - Add JSON tags for proper serialization
  - _Requirements: 1.1, 2.1, 3.1, 6.1_

- [x] 3. Implement FileService for temporary storage

  - Create FileService with SaveUpload, GetFile, DeleteFile methods
  - Implement UUID generation for unique file identifiers
  - Add file path management for temporary storage
  - Implement ScheduleCleanup with goroutine-based cleanup scheduling
  - _Requirements: 6.1, 6.2_

- [x] 3.1 Write property test for unique file identifiers

  - **Property 16: Unique file identifiers**
  - **Validates: Requirements 6.1**

- [x] 4. Implement ImageValidationService

  - Create ValidateFormat method to check JPEG, PNG, WebP using magic numbers
  - Create ValidateSize method to enforce 10MB limit
  - Implement DetectHumanFeatures method (basic implementation or placeholder)
  - _Requirements: 1.1, 1.2, 1.4, 1.5, 2.1, 2.2, 2.4_

- [x] 4.1 Write property test for valid image format acceptance

  - **Property 1: Valid image format acceptance**
  - **Validates: Requirements 1.1, 2.1**

- [x] 4.2 Write property test for oversized image rejection

  - **Property 2: Oversized image rejection**
  - **Validates: Requirements 1.2, 2.2**

- [x] 4.3 Write property test for invalid format rejection

  - **Property 4: Invalid format rejection**
  - **Validates: Requirements 1.4, 2.4**

- [x] 5. Implement upload API endpoints

  - Create POST /api/upload/user-photo endpoint with multipart handling
  - Create POST /api/upload/shirt endpoint with multipart handling
  - Integrate FileService and ImageValidationService
  - Return proper JSON responses with fileId and previewUrl
  - Implement error handling for validation failures
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, 2.4_

- [x] 5.1 Write property test for successful upload preview

  - **Property 3: Successful upload preview**
  - **Validates: Requirements 1.3, 2.3**

- [x] 6. Implement preview endpoint

  - Create GET /api/preview/:fileId endpoint
  - Serve image files with proper Content-Type headers
  - Handle missing file errors
  - _Requirements: 1.3, 2.3_

- [x] 7. Implement AIService client

  - Create AIService struct with HTTP client
  - Implement SubmitTryOnJob method to call AI API
  - Implement CheckJobStatus method for polling
  - Implement GetResult method to retrieve final image
  - Add authentication header handling
  - Implement retry logic with exponential backoff (3 attempts)
  - _Requirements: 3.2, 8.1, 8.2, 8.3, 8.4, 8.5_

- [x] 7.1 Write property test for AI service request formatting

  - **Property 21: AI service request formatting**
  - **Validates: Requirements 8.1**

- [x] 7.2 Write property test for AI service authentication

  - **Property 22: AI service authentication**
  - **Validates: Requirements 8.2**

-

- [x] 7.3 Write property test for AI service response parsing

  - **Property 23: AI service response parsing**
  - **Validates: Requirements 8.3**

-

- [x] 7.4 Write property test for retry with exponential backoff

  - **Property 24: Retry with exponential backoff**
  - **Validates: Requirements 8.4**

-

- [x] 7.5 Write property test for exhausted retry notification

  - **Property 25: Exhausted retry notification**
  - **Validates: Requirements 8.5**

- [x] 8. Implement processing job management

  - Create in-memory job store (map with mutex)
  - Implement job creation and status tracking
  - Add methods to update job progress and status
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

-

- [x] 9. Implement processing API endpoints

  - Create POST /api/process endpoint
  - Validate that both user photo and shirt image exist
  - Create processing job and initiate AI service call asynchronously
  - Return jobId and estimated time
  - _Requirements: 3.1, 3.2, 7.3_

- [x] 9.1 Write property test for image forwarding to AI service

  - **Property 6: Image forwarding to AI service**
  - **Validates: Requirements 3.2**

- [x] 9.2 Write property test for processing time estimation

  - **Property 19: Processing time estimation**
  - **Validates: Requirements 7.3**

- [x] 10. Implement status polling endpoint

  - Create GET /api/status/:jobId endpoint
  - Return current job status, progress, and message
  - Include resultUrl when job is completed
  - _Requirements: 3.3, 3.4, 3.5, 7.4_

- [x] 11. Implement result download endpoint

  - Create GET /api/result/:resultId endpoint
  - Serve composite image with proper headers
  - Set Content-Disposition for download with descriptive filename
  - _Requirements: 4.1, 4.2, 4.3_

- [x] 11.1 Write property test for download with proper filename

  - **Property 11: Download with proper filename**
  - **Validates: Requirements 4.2**

- [x] 11.2 Write property test for image quality preservation

  - **Property 12: Image quality preservation**
  - **Validates: Requirements 4.3**

- [x] 12. Implement error handling and logging

  - Add structured logging throughout backend
  - Implement consistent error response format
  - Add specific error messages for different failure types
  - _Requirements: 7.5_

-

- [x] 12.1 Write property test for error message specificity

  - **Property 20: Error message specificity**
  - **Validates: Requirements 7.5**

- [x] 13. Set up frontend React application

  - Initialize React app with TypeScript
  - Set up basic routing and app structure
  - Configure API client for backend communication
  - _Requirements: All_

- [x] 14. Implement UploadInterface component

  - Create file input with drag-and-drop support
  - Implement client-side validation (file type, size)
  - Display image preview after upload
  - Show upload progress indicator
  - Handle upload errors with user-friendly messages
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, 2.4, 7.1, 7.2_

- [x] 14.1 Write property test for upload progress feedback

  - **Property 17: Upload progress feedback**
  - **Validates: Requirements 7.1**

- [x] 14.2 Write property test for success confirmation

  - **Property 18: Success confirmation**
  - **Validates: Requirements 7.2**

- [x] 15. Implement ProcessingView component

- [ ] 15. Implement ProcessingView component

  - Create loading indicator with animation
  - Display status messages from backend
  - Show estimated processing time
  - Update status every 5 seconds via polling
  - _Requirements: 3.3, 7.3, 7.4_

- [x] 15.1 Write property test for loading state display

  - **Property 7: Loading state display**
  - **Validates: Requirements 3.3**

- [x] 16. Implement ResultDisplay component

  - Display composite image when processing completes
  - Add download button with proper filename
  - Add "Try Another Shirt" button
  - _Requirements: 3.4, 4.1, 4.2, 5.1_

- [x] 16.1 Write property test for successful result display

  - **Property 8: Successful result display**
  - **Validates: Requirements 3.4**

- [x] 16.2 Write property test for download availability

  - **Property 10: Download availability**
  - **Validates: Requirements 4.1**

- [x] 17. Implement main App component with state management

  - Create application state for user photo, shirt image, and result
  - Implement upload handlers for both image types
  - Implement process initiation logic
  - Implement status polling logic
  - Handle error states and display error messages
  - _Requirements: 3.1, 3.5, 5.2, 5.3, 7.5_

- [x] 17.1 Write property test for processing enablement

  - **Property 5: Processing enablement**
  - **Validates: Requirements 3.1**

- [x] 17.2 Write property test for error handling with retry

  - **Property 9: Error handling with retry**
  - **Validates: Requirements 3.5**

- [x] 17.3 Write property test for shirt replacement capability

  - **Property 13: Shirt replacement capability**
  - **Validates: Requirements 5.1**

- [x] 17.4 Write property test for user photo persistence

  - **Property 14: User photo persistence**
  - **Validates: Requirements 5.2**

- [x] 17.5 Write property test for reprocessing without re-upload

  - **Property 15: Reprocessing without re-upload**
  - **Validates: Requirements 5.3**

- [x] 18. Add styling and responsive design

  - Create CSS for all components
  - Ensure mobile responsiveness
  - Add loading animations and transitions
  - Implement clean, user-friendly UI
  - _Requirements: All_

- [x] 19. Implement CORS and security headers

  - Configure CORS middleware in Go backend
  - Add security headers (HTTPS enforcement, CSP)
  - Implement rate limiting middleware
  - _Requirements: 6.3, 6.4_

- [x] 20. Add configuration and environment variables

  - Create config file for backend (AI API key, port, storage path)
  - Create environment variable handling
  - Add frontend environment config for API URL
  - _Requirements: 8.2_

- [x] 21. Checkpoint - Ensure all tests pass

  - Ensure all tests pass, ask the user if questions arise.

- [x] 22. Create README and documentation

  - Write setup instructions for both frontend and backend
  - Document API endpoints
  - Add environment variable documentation
  - Include example usage and screenshots
  - _Requirements: All_

- [x] 23. Implement email notification system for long processing times

  - Add email field to ProcessingJob model
  - Create EmailService with SendCompletionEmail and ValidateEmail methods
  - Integrate email service (SMTP or API like SendGrid)
  - Generate secure time-limited result URLs
  - _Requirements: 9.2, 9.3, 9.4_

- [x] 23.1 Write property test for email collection and background processing

  - **Property 28: Email collection and background processing**
  - **Validates: Requirements 9.3**

- [x] 23.2 Write property test for email delivery on completion

  - **Property 29: Email delivery on completion**
  - **Validates: Requirements 9.4**

- [x] 24. Implement email notification API endpoint

  - Create POST /api/notify-email endpoint
  - Validate email address format
  - Associate email with job ID
  - Return success confirmation
  - _Requirements: 9.2, 9.3_

- [x] 25. Update ProcessingView component for long processing times

  - Track processing start time and elapsed duration
  - Detect when processing exceeds 3 minutes
  - Display notification modal with email option
  - Add email input form with validation
  - Add "Continue Waiting" button
  - Handle email submission to backend
  - _Requirements: 9.1, 9.2, 9.5_

- [x] 25.1 Write property test for long processing notification

  - **Property 26: Long processing notification**
  - **Validates: Requirements 9.1**

- [x] 25.2 Write property test for email notification option availability

  - **Property 27: Email notification option availability**
  - **Validates: Requirements 9.2**

- [x] 25.3 Write property test for continued progress updates

  - **Property 30: Continued progress updates**
  - **Validates: Requirements 9.5**

- [x] 26. Update job processing to send emails on completion

  - Check if job has associated email when completing
  - Send completion email with result link
  - Log email sending status
  - Handle email sending failures gracefully
  - _Requirements: 9.4_

- [x] 27. Create email templates

  - Design HTML email template for completion notification
  - Include result link with expiration notice
  - Add plain text fallback
  - Test email rendering across clients
  - _Requirements: 9.4_

- [x] 28. Implement result access via email link

  - Create GET /api/result-link/:token endpoint
  - Validate token and check expiration (24 hours)
  - Serve result page or redirect to result
  - Handle expired links gracefully
  - _Requirements: 9.4_

- [x] 29. Add configuration for email service

  - Add email service credentials to environment variables
  - Configure SMTP settings or API keys
  - Add result URL base configuration
  - Document email setup in README
  - _Requirements: 9.3, 9.4_

- [x] 30. Final checkpoint - Test long processing flow

  - Ensure all tests pass, ask the user if questions arise.
  - Test complete flow: upload → long wait → email notification → result access
