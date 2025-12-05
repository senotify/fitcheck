# Design Document

## Overview

The Virtual FitCheck system is a full-stack web application that enables users to virtually try on clothing using generative AI. The system consists of a frontend web interface for image uploads and result display, a backend API for request handling and orchestration, and integration with a generative AI service for image composition.

The architecture follows a client-server model where the frontend handles user interactions and image previews, while the backend manages file uploads, coordinates with the AI service, and handles temporary storage. The system is designed to be stateless, with temporary file storage and no permanent user data retention.

## Architecture

### High-Level Architecture

```
┌─────────────────┐
│   Web Browser   │
│   (Frontend)    │
└────────┬────────┘
         │ HTTPS
         ▼
┌─────────────────┐
│   Backend API   │
│      (Go)       │
└────────┬────────┘
         │
         ├─────────────┐
         │             │
         ▼             ▼
┌──────────────┐  ┌──────────────┐
│  Temporary   │  │ Generative   │
│  File Store  │  │  AI Service  │
└──────────────┘  └──────────────┘
```

### Technology Stack

**Frontend:**

- React with TypeScript for UI components
- HTML5 File API for image uploads
- Fetch API for backend communication
- CSS for styling and responsive design

**Backend:**

- Go with Gin or Chi framework for HTTP routing
- Go standard library multipart for file upload handling
- Go image libraries (image, image/jpeg, image/png) for validation and preprocessing
- Go HTTP client for requests to AI service
- Go standard library (os, ioutil) for temporary file storage

**AI Integration:**

- Replicate API or similar generative AI service
- Virtual try-on models (e.g., Stable Diffusion with ControlNet, or specialized try-on models)

## Components and Interfaces

### Frontend Components

#### 1. UploadInterface Component

Handles user photo and shirt image uploads with drag-and-drop and file selection.

**Props:**

- `onUpload: (file: File, type: 'user' | 'shirt') => void`
- `uploadProgress: number`
- `previewUrl: string | null`

**Responsibilities:**

- Validate file types and sizes client-side
- Display image previews
- Show upload progress
- Emit upload events to parent component

#### 2. ProcessingView Component

Displays processing status and loading indicators.

**Props:**

- `isProcessing: boolean`
- `statusMessage: string`
- `estimatedTime: number`
- `processingDuration: number`
- `onEmailNotification: (email: string) => void`
- `onContinueWaiting: () => void`

**Responsibilities:**

- Show animated loading indicator
- Display status updates
- Show estimated completion time
- Detect when processing exceeds 3 minutes
- Display long processing notification with email option
- Handle email input and submission
- Allow user to choose to continue waiting

#### 3. ResultDisplay Component

Shows the generated composite image with download options.

**Props:**

- `compositeImageUrl: string`
- `onDownload: () => void`
- `onTryAnother: () => void`

**Responsibilities:**

- Display final composite image
- Provide download functionality
- Allow user to try another shirt

#### 4. App Component (Main)

Orchestrates the entire user flow and manages application state.

**State:**

- `userPhoto: File | null`
- `shirtImage: File | null`
- `compositeResult: string | null`
- `processingStatus: ProcessingStatus`
- `error: string | null`

### Backend API Endpoints

#### POST /api/upload/user-photo

Accepts user photo upload.

**Request:**

- Content-Type: multipart/form-data
- Body: image file

**Response:**

```json
{
  "success": true,
  "fileId": "uuid-v4-string",
  "previewUrl": "/api/preview/uuid-v4-string"
}
```

#### POST /api/upload/shirt

Accepts shirt image upload.

**Request:**

- Content-Type: multipart/form-data
- Body: image file

**Response:**

```json
{
  "success": true,
  "fileId": "uuid-v4-string",
  "previewUrl": "/api/preview/uuid-v4-string"
}
```

#### POST /api/process

Initiates virtual try-on processing.

**Request:**

```json
{
  "userPhotoId": "uuid-string",
  "shirtImageId": "uuid-string"
}
```

**Response:**

```json
{
  "success": true,
  "jobId": "uuid-string",
  "estimatedTime": 15
}
```

#### GET /api/status/:jobId

Polls for processing status.

**Response:**

```json
{
  "status": "processing" | "completed" | "failed",
  "progress": 75,
  "message": "Applying shirt to image...",
  "resultUrl": "/api/result/uuid-string" // only when completed
}
```

#### GET /api/result/:resultId

Downloads the composite image.

**Response:**

- Content-Type: image/jpeg
- Body: image binary data

#### POST /api/notify-email

Registers email notification for a long-running job.

**Request:**

```json
{
  "jobId": "uuid-string",
  "email": "user@example.com"
}
```

**Response:**

```json
{
  "success": true,
  "message": "You will receive an email when processing completes"
}
```

### Backend Services

#### FileService

Manages temporary file storage and cleanup.

**Methods:**

- `SaveUpload(file []byte, fileId string) (string, error)`
- `GetFile(fileId string) ([]byte, error)`
- `DeleteFile(fileId string) error`
- `ScheduleCleanup(fileId string, delay time.Duration)`

#### ImageValidationService

Validates uploaded images.

**Methods:**

- `ValidateFormat(file []byte) (bool, error)`
- `ValidateSize(file []byte) (bool, error)`
- `DetectHumanFeatures(file []byte) (bool, error)`

#### AIService

Interfaces with the generative AI API.

**Methods:**

- `SubmitTryOnJob(userPhoto []byte, shirtImage []byte) (string, error)`
- `CheckJobStatus(jobId string) (*JobStatus, error)`
- `GetResult(jobId string) ([]byte, error)`

#### EmailService

Handles email notifications for completed jobs.

**Methods:**

- `SendCompletionEmail(email string, resultUrl string, jobId string) error`
- `ValidateEmail(email string) (bool, error)`

## Data Models

### File Upload

```go
type UploadedFile struct {
  FileID       string    `json:"fileId"`
  OriginalName string    `json:"originalName"`
  MimeType     string    `json:"mimeType"`
  Size         int64     `json:"size"`
  UploadedAt   time.Time `json:"uploadedAt"`
  ExpiresAt    time.Time `json:"expiresAt"`
}
```

### Processing Job

```go
type JobStatus string

const (
  JobStatusPending    JobStatus = "pending"
  JobStatusProcessing JobStatus = "processing"
  JobStatusCompleted  JobStatus = "completed"
  JobStatusFailed     JobStatus = "failed"
)

type ProcessingJob struct {
  JobID         string     `json:"jobId"`
  UserPhotoID   string     `json:"userPhotoId"`
  ShirtImageID  string     `json:"shirtImageId"`
  Status        JobStatus  `json:"status"`
  Progress      int        `json:"progress"`
  StatusMessage string     `json:"statusMessage"`
  ResultID      *string    `json:"resultId,omitempty"`
  Error         *string    `json:"error,omitempty"`
  CreatedAt     time.Time  `json:"createdAt"`
  CompletedAt   *time.Time `json:"completedAt,omitempty"`
}
```

### AI Service Request

```go
type AITryOnRequest struct {
  PersonImage  string           `json:"personImage"`  // base64 encoded
  GarmentImage string           `json:"garmentImage"` // base64 encoded
  Options      *AIRequestOptions `json:"options,omitempty"`
}

type AIRequestOptions struct {
  Quality      string `json:"quality"`      // "standard" or "high"
  PreserveFace bool   `json:"preserveFace"`
}
```

### AI Service Response

```go
type AITryOnResponse struct {
  ID     string  `json:"id"`
  Status string  `json:"status"` // "starting", "processing", "succeeded", "failed"
  Output *string `json:"output,omitempty"` // URL or base64 of result
  Error  *string `json:"error,omitempty"`
}
```

### Email Notification Request

```go
type EmailNotificationRequest struct {
  JobID string `json:"jobId"`
  Email string `json:"email"`
}

type EmailNotificationResponse struct {
  Success bool   `json:"success"`
  Message string `json:"message"`
}
```

## Correctness Properties

_A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees._

### Property 1: Valid image format acceptance

_For any_ uploaded image file with format JPEG, PNG, or WebP, the upload validation should accept the file regardless of whether it's a user photo or shirt image.
**Validates: Requirements 1.1, 2.1**

### Property 2: Oversized image rejection

_For any_ uploaded image file exceeding 10MB, the upload validation should reject the file and return an error, regardless of whether it's a user photo or shirt image.
**Validates: Requirements 1.2, 2.2**

### Property 3: Successful upload preview

_For any_ successfully uploaded image, the system should return a valid preview URL that can be used to display the image.
**Validates: Requirements 1.3, 2.3**

### Property 4: Invalid format rejection

_For any_ uploaded file with an invalid image format (not JPEG, PNG, or WebP), the upload validation should reject the file and return an error message listing acceptable formats.
**Validates: Requirements 1.4, 2.4**

### Property 5: Processing enablement

_For any_ application state, the processing action should be enabled if and only if both a user photo and shirt image have been successfully uploaded.
**Validates: Requirements 3.1**

### Property 6: Image forwarding to AI service

_For any_ processing request with valid user photo and shirt image IDs, the backend should send both complete images to the Generative AI Service.
**Validates: Requirements 3.2**

### Property 7: Loading state display

_For any_ processing job with status "processing", the UI should display a loading indicator.
**Validates: Requirements 3.3**

### Property 8: Successful result display

_For any_ processing job that completes with status "completed", the system should display the composite image to the user.
**Validates: Requirements 3.4**

### Property 9: Error handling with retry

_For any_ processing job that fails, the system should display an error message and maintain the ability to retry processing with the same images.
**Validates: Requirements 3.5**

### Property 10: Download availability

_For any_ successfully generated composite image, the system should provide a download button in the UI.
**Validates: Requirements 4.1**

### Property 11: Download with proper filename

_For any_ composite image download action, the system should initiate a file download with a filename that includes a timestamp or unique identifier.
**Validates: Requirements 4.2**

### Property 12: Image quality preservation

_For any_ generated composite image, the downloaded file should have identical dimensions and comparable quality to the generated result (no additional lossy compression).
**Validates: Requirements 4.3**

### Property 13: Shirt replacement capability

_For any_ application state with a generated composite image, the system should allow uploading a new shirt image without clearing the existing user photo.
**Validates: Requirements 5.1**

### Property 14: User photo persistence

_For any_ new shirt image upload after initial processing, the user photo ID should remain unchanged from the previous processing session.
**Validates: Requirements 5.2**

### Property 15: Reprocessing without re-upload

_For any_ processing request using a previously uploaded user photo, the system should successfully generate a new composite without requiring the user photo to be uploaded again.
**Validates: Requirements 5.3**

### Property 16: Unique file identifiers

_For any_ set of uploaded images, each image should receive a unique identifier, with no two uploads sharing the same ID.
**Validates: Requirements 6.1**

### Property 17: Upload progress feedback

_For any_ image upload in progress, the system should emit progress updates with percentage values between 0 and 100.
**Validates: Requirements 7.1**

### Property 18: Success confirmation

_For any_ completed upload, the system should display a success message to the user.
**Validates: Requirements 7.2**

### Property 19: Processing time estimation

_For any_ initiated processing request, the response should include an estimated processing time in seconds.
**Validates: Requirements 7.3**

### Property 20: Error message specificity

_For any_ error condition (upload failure, validation failure, processing failure), the system should display an error message that specifically describes the issue rather than a generic error.
**Validates: Requirements 7.5**

### Property 21: AI service request formatting

_For any_ processing request sent to the Generative AI Service, the request payload should conform to the AI service's API specification with properly formatted image data and required fields.
**Validates: Requirements 8.1**

### Property 22: AI service authentication

_For any_ request to the Generative AI Service, the HTTP request should include valid authentication credentials in the headers.
**Validates: Requirements 8.2**

### Property 23: AI service response parsing

_For any_ valid response from the Generative AI Service, the system should successfully parse the response and extract the result URL or image data.
**Validates: Requirements 8.3**

### Property 24: Retry with exponential backoff

_For any_ failed request to the Generative AI Service, the system should retry up to 3 times with exponentially increasing delays between attempts.
**Validates: Requirements 8.4**

### Property 25: Exhausted retry notification

_For any_ processing request where all 3 retry attempts to the AI service fail, the system should log the error and notify the user that the service is temporarily unavailable.
**Validates: Requirements 8.5**

### Property 26: Long processing notification

_For any_ processing job that exceeds 3 minutes, the system should display a notification to the user informing them that processing is taking longer than expected.
**Validates: Requirements 9.1**

### Property 27: Email notification option availability

_For any_ processing job that exceeds 3 minutes, the system should present the user with options to either receive results via email or continue waiting.
**Validates: Requirements 9.2**

### Property 28: Email collection and background processing

_For any_ user who chooses email notification, the system should collect a valid email address and continue processing the job in the background without requiring the user to remain on the page.
**Validates: Requirements 9.3**

### Property 29: Email delivery on completion

_For any_ background processing job that completes, the system should send an email to the provided address containing a link to view the composite image.
**Validates: Requirements 9.4**

### Property 30: Continued progress updates

_For any_ user who chooses to continue waiting beyond 3 minutes, the system should continue displaying progress updates at regular intervals.
**Validates: Requirements 9.5**

## Error Handling

### Client-Side Error Handling

**File Upload Errors:**

- Invalid file type: Display error with list of accepted formats
- File too large: Display error with maximum size limit
- Network failure: Allow retry with exponential backoff
- Validation failure: Display specific validation error message

**Processing Errors:**

- AI service unavailable: Display user-friendly message with retry option
- Timeout: Display timeout message and allow retry
- Invalid response: Log error details and show generic error to user

### Server-Side Error Handling

**Upload Endpoint Errors:**

- Malformed request: Return 400 Bad Request with error details
- File validation failure: Return 422 Unprocessable Entity with validation errors
- Storage failure: Return 500 Internal Server Error and log details
- Rate limiting: Return 429 Too Many Requests

**Processing Endpoint Errors:**

- Missing files: Return 404 Not Found with specific file ID
- AI service error: Return 502 Bad Gateway and log AI service response
- Timeout: Return 504 Gateway Timeout after retry attempts exhausted
- Invalid job ID: Return 404 Not Found

**Error Response Format:**

```go
type ErrorResponse struct {
  Success bool       `json:"success"`
  Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
  Code    string      `json:"code"`
  Message string      `json:"message"`
  Details interface{} `json:"details,omitempty"`
}
```

### Cleanup and Resource Management

**Automatic Cleanup:**

- Schedule file deletion 1 hour after upload
- Clean up orphaned files (no associated job) after 2 hours
- Cancel and clean up jobs that exceed 5-minute processing time
- Implement graceful shutdown to complete in-flight requests

**Manual Cleanup:**

- Provide admin endpoint to manually trigger cleanup
- Log all cleanup operations for audit trail

## Testing Strategy

### Unit Testing

The system will use Jest for frontend unit tests and Go's testing package for backend unit tests.

**Frontend Unit Tests:**

- Component rendering with different props
- File validation logic (format, size)
- State management and transitions
- Event handler behavior
- Error message display

**Backend Unit Tests (Go):**

- API endpoint request/response handling using httptest
- File service operations (save, retrieve, delete)
- Image validation logic
- AI service client methods
- Error handling and retry logic

**Example Unit Tests:**

- Test that UploadInterface rejects files over 10MB
- Test that FileService generates unique IDs
- Test that AIService formats requests correctly
- Test error response format for various failure scenarios

### Property-Based Testing

The system will use fast-check for property-based testing in JavaScript/TypeScript for frontend tests, and gopter for property-based testing in Go for backend tests.

**Configuration:**

- Each property test should run a minimum of 100 iterations
- Use appropriate generators for test data (images, file sizes, formats)
- Tag each test with the correctness property it validates

**Property Test Requirements:**

- Each property-based test MUST include a comment with the format: `// Feature: virtual-fitcheck, Property X: [property description]`
- Each correctness property from this document MUST be implemented by a SINGLE property-based test
- Tests should use realistic generators that produce valid test data within expected ranges
- Frontend property tests use fast-check (TypeScript)
- Backend property tests use gopter (Go)

**Example Property Tests:**

- Generate random valid image formats and verify acceptance (Property 1) - Go backend test
- Generate random file sizes and verify rejection over 10MB (Property 2) - Go backend test
- Generate random upload states and verify processing button enablement (Property 5) - TypeScript frontend test
- Generate random error conditions and verify specific error messages (Property 20) - Both frontend and backend tests

### Integration Testing

**API Integration Tests:**

- Full upload-to-processing-to-download flow
- Error recovery scenarios
- Concurrent upload handling
- File cleanup verification

**AI Service Integration Tests:**

- Mock AI service responses for testing
- Test retry logic with simulated failures
- Verify request/response format compatibility

### Test Data Generators

**Image Generators:**

- Generate valid JPEG, PNG, WebP files of various sizes
- Generate invalid file formats
- Generate images with and without human features
- Generate oversized files (>10MB)

**State Generators:**

- Generate various application states (no uploads, one upload, both uploads, processing, completed)
- Generate error states
- Generate job status transitions

## Implementation Notes

### AI Service Selection

The system should integrate with a virtual try-on AI service. Recommended options:

- Replicate API with virtual try-on models
- Stable Diffusion with ControlNet for pose preservation
- Specialized try-on APIs like Fashn AI or Revery AI

The AIService should be designed with an interface that allows swapping AI providers without changing other components.

### Performance Considerations

**Image Optimization:**

- Resize large images before sending to AI service (max 1024x1024)
- Use WebP format for previews to reduce bandwidth
- Implement lazy loading for result images

**Caching:**

- Cache user photos for 1 hour to enable multiple shirt try-ons
- Cache AI service results for 1 hour to avoid reprocessing identical requests

**Async Processing:**

- Use job queue for processing requests
- Implement polling with exponential backoff (1s, 2s, 4s, 8s intervals)
- Set maximum processing timeout of 5 minutes

### Security Considerations

**Input Validation:**

- Validate file types using magic numbers, not just extensions
- Scan uploaded files for malware
- Limit upload rate per IP address

**Data Privacy:**

- Do not log or store user images permanently
- Use secure random UUIDs for file identifiers
- Implement HTTPS for all communications
- Clear temporary files on schedule
- Store email addresses only for active jobs, delete after notification sent

**API Security:**

- Implement rate limiting (e.g., 10 requests per minute per IP)
- Use CORS to restrict frontend origins
- Validate all request parameters
- Sanitize error messages to avoid information leakage
- Validate email addresses to prevent injection attacks

### Long Processing Time Handling

**Email Notification System:**

- Use a simple SMTP service or email API (e.g., SendGrid, AWS SES, Mailgun)
- Store email address with job record when user opts for notification
- Generate secure, time-limited links for result access (valid for 24 hours)
- Send email immediately when job completes
- Include job ID and timestamp in email for tracking

**Background Job Processing:**

- Continue processing jobs even after user closes browser
- Store job state persistently (consider using Redis or database for production)
- Implement job timeout of 15 minutes maximum
- Clean up completed jobs and associated data after 24 hours

**Frontend Timeout Detection:**

- Track processing start time in component state
- Check elapsed time on each status poll
- Display notification modal when 3 minutes exceeded
- Provide clear options: "Get Email Notification" or "Keep Waiting"
- Allow user to leave page after email submission

**Email Template:**

- Subject: "Your Virtual FitCheck is Ready!"
- Include direct link to result page
- Add expiration notice (24 hours)
- Include support contact information
