package models

import "time"

// JobStatus represents the current status of a processing job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// ProcessingJob represents a virtual try-on processing job
type ProcessingJob struct {
	JobID         string     `json:"jobId" db:"id"`
	SessionID     string     `json:"sessionId" db:"session_id"`
	UserPhotoID   string     `json:"userPhotoId" db:"user_photo_id"`
	ShirtImageID  string     `json:"shirtImageId" db:"shirt_image_id"`
	Status        JobStatus  `json:"status" db:"status"`
	Progress      int        `json:"progress" db:"progress"`
	StatusMessage string     `json:"statusMessage" db:"status_message"`
	ResultID      *string    `json:"resultId,omitempty" db:"result_id"`
	Error         *string    `json:"error,omitempty" db:"error"`
	Email         *string    `json:"email,omitempty" db:"email"`
	ResultToken   *string    `json:"resultToken,omitempty" db:"result_token"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
	CompletedAt   *time.Time `json:"completedAt,omitempty" db:"completed_at"`
}
