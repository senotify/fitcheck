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
	JobID         string     `json:"jobId"`
	UserPhotoID   string     `json:"userPhotoId"`
	ShirtImageID  string     `json:"shirtImageId"`
	Status        JobStatus  `json:"status"`
	Progress      int        `json:"progress"`
	StatusMessage string     `json:"statusMessage"`
	ResultID      *string    `json:"resultId,omitempty"`
	Error         *string    `json:"error,omitempty"`
	Email         *string    `json:"email,omitempty"`
	ResultToken   *string    `json:"resultToken,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
}
