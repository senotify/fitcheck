package models

import "time"

// UploadedFile represents a file uploaded by a user
type UploadedFile struct {
	FileID       string    `json:"fileId"`
	OriginalName string    `json:"originalName"`
	MimeType     string    `json:"mimeType"`
	Size         int64     `json:"size"`
	UploadedAt   time.Time `json:"uploadedAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}
