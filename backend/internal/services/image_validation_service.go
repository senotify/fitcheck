package services

import (
	"bytes"
	"fmt"
)

// ImageValidationService validates uploaded images
type ImageValidationService struct {
	maxSizeBytes int64
}

// NewImageValidationService creates a new ImageValidationService instance
func NewImageValidationService(maxSizeMB int) *ImageValidationService {
	return &ImageValidationService{
		maxSizeBytes: int64(maxSizeMB) * 1024 * 1024,
	}
}

// Magic numbers for image format detection
var (
	jpegMagic = [][]byte{
		{0xFF, 0xD8, 0xFF}, // JPEG
	}
	pngMagic = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG
	webpMagic = []byte{0x52, 0x49, 0x46, 0x46}                         // RIFF (WebP container)
	webpSig   = []byte{0x57, 0x45, 0x42, 0x50}                         // WEBP signature at offset 8
)

// ValidateFormat checks if the file is a valid JPEG, PNG, or WebP using magic numbers
func (ivs *ImageValidationService) ValidateFormat(file []byte) (bool, error) {
	if len(file) < 12 {
		return false, fmt.Errorf("file too small to determine format")
	}

	// Check JPEG
	for _, magic := range jpegMagic {
		if bytes.HasPrefix(file, magic) {
			return true, nil
		}
	}

	// Check PNG
	if bytes.HasPrefix(file, pngMagic) {
		return true, nil
	}

	// Check WebP (RIFF container with WEBP signature)
	if bytes.HasPrefix(file, webpMagic) && len(file) >= 12 {
		if bytes.Equal(file[8:12], webpSig) {
			return true, nil
		}
	}

	return false, fmt.Errorf("invalid image format: only JPEG, PNG, and WebP are accepted")
}

// ValidateSize checks if the file size is within the allowed limit
func (ivs *ImageValidationService) ValidateSize(file []byte) (bool, error) {
	fileSize := int64(len(file))
	
	if fileSize > ivs.maxSizeBytes {
		return false, fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes (%.1f MB)", 
			fileSize, ivs.maxSizeBytes, float64(ivs.maxSizeBytes)/(1024*1024))
	}

	return true, nil
}

// DetectHumanFeatures is a placeholder for human feature detection
// In a production system, this would use computer vision libraries or AI services
func (ivs *ImageValidationService) DetectHumanFeatures(file []byte) (bool, error) {
	// Basic implementation: just check if it's a valid image format
	// A real implementation would use libraries like:
	// - OpenCV for face detection
	// - TensorFlow/PyTorch models for human pose detection
	// - Cloud vision APIs (Google Vision, AWS Rekognition, etc.)
	
	isValid, err := ivs.ValidateFormat(file)
	if err != nil {
		return false, fmt.Errorf("cannot detect features in invalid image: %w", err)
	}

	if !isValid {
		return false, fmt.Errorf("image format not supported for feature detection")
	}

	// Placeholder: assume all valid images contain human features
	// TODO: Implement actual human feature detection
	return true, nil
}
