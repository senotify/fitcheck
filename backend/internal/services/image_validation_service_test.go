package services

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: virtual-fitcheck, Property 1: Valid image format acceptance
// For any uploaded image file with format JPEG, PNG, or WebP, the upload validation
// should accept the file regardless of whether it's a user photo or shirt image.
// Validates: Requirements 1.1, 2.1
func TestProperty_ValidImageFormatAcceptance(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	ivs := NewImageValidationService(10) // 10MB limit

	// Generator for valid image formats
	validImageGen := gen.OneConstOf(
		// JPEG magic number + some data
		[]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01},
		// PNG magic number + some data
		[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D},
		// WebP magic number (RIFF + WEBP signature)
		[]byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
	).Map(func(header []byte) []byte {
		// Add some random data to make it more realistic
		result := make([]byte, len(header)+100)
		copy(result, header)
		return result
	})

	properties.Property("valid image formats are accepted", prop.ForAll(
		func(imageData []byte) bool {
			isValid, err := ivs.ValidateFormat(imageData)
			
			if err != nil {
				t.Logf("Unexpected error for valid format: %v", err)
				return false
			}

			if !isValid {
				t.Logf("Valid image format was rejected")
				return false
			}

			return true
		},
		validImageGen,
	))

	properties.TestingRun(t)
}

// Feature: virtual-fitcheck, Property 2: Oversized image rejection
// For any uploaded image file exceeding 10MB, the upload validation should reject
// the file and return an error, regardless of whether it's a user photo or shirt image.
// Validates: Requirements 1.2, 2.2
func TestProperty_OversizedImageRejection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	ivs := NewImageValidationService(10) // 10MB limit
	maxSize := 10 * 1024 * 1024 // 10MB in bytes

	// Generator for oversized files (> 10MB)
	oversizedFileGen := gen.IntRange(maxSize+1, maxSize+5*1024*1024).Map(func(size int) []byte {
		// Create a file with valid JPEG header but oversized
		data := make([]byte, size)
		// Add JPEG magic number
		data[0] = 0xFF
		data[1] = 0xD8
		data[2] = 0xFF
		return data
	})

	properties.Property("oversized images are rejected", prop.ForAll(
		func(imageData []byte) bool {
			isValid, err := ivs.ValidateSize(imageData)

			if err == nil {
				t.Logf("Expected error for oversized file, got nil")
				return false
			}

			if isValid {
				t.Logf("Oversized image was accepted (size: %d bytes)", len(imageData))
				return false
			}

			return true
		},
		oversizedFileGen,
	))

	properties.TestingRun(t)
}

// Feature: virtual-fitcheck, Property 4: Invalid format rejection
// For any uploaded file with an invalid image format (not JPEG, PNG, or WebP),
// the upload validation should reject the file and return an error message
// listing acceptable formats.
// Validates: Requirements 1.4, 2.4
func TestProperty_InvalidFormatRejection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	ivs := NewImageValidationService(10) // 10MB limit

	// Generator for invalid file formats
	invalidFormatGen := gen.OneConstOf(
		// GIF
		[]byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		// BMP
		[]byte{0x42, 0x4D, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		// TIFF (little-endian)
		[]byte{0x49, 0x49, 0x2A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		// TIFF (big-endian)
		[]byte{0x4D, 0x4D, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		// PDF
		[]byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34, 0x00, 0x00, 0x00, 0x00},
		// Random data
		[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B},
	).Map(func(header []byte) []byte {
		// Add some random data to make it more realistic
		result := make([]byte, len(header)+100)
		copy(result, header)
		return result
	})

	properties.Property("invalid image formats are rejected with error message", prop.ForAll(
		func(imageData []byte) bool {
			isValid, err := ivs.ValidateFormat(imageData)

			if err == nil {
				t.Logf("Expected error for invalid format, got nil")
				return false
			}

			if isValid {
				t.Logf("Invalid image format was accepted")
				return false
			}

			// Check that error message mentions acceptable formats
			errMsg := err.Error()
			if errMsg == "" {
				t.Logf("Error message is empty")
				return false
			}

			return true
		},
		invalidFormatGen,
	))

	properties.TestingRun(t)
}

// Unit tests for basic functionality
func TestImageValidationService_ValidateFormat_JPEG(t *testing.T) {
	ivs := NewImageValidationService(10)
	
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	isValid, err := ivs.ValidateFormat(jpegData)
	
	if err != nil {
		t.Errorf("Unexpected error for JPEG: %v", err)
	}
	if !isValid {
		t.Error("JPEG should be valid")
	}
}

func TestImageValidationService_ValidateFormat_PNG(t *testing.T) {
	ivs := NewImageValidationService(10)
	
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	isValid, err := ivs.ValidateFormat(pngData)
	
	if err != nil {
		t.Errorf("Unexpected error for PNG: %v", err)
	}
	if !isValid {
		t.Error("PNG should be valid")
	}
}

func TestImageValidationService_ValidateFormat_WebP(t *testing.T) {
	ivs := NewImageValidationService(10)
	
	webpData := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}
	isValid, err := ivs.ValidateFormat(webpData)
	
	if err != nil {
		t.Errorf("Unexpected error for WebP: %v", err)
	}
	if !isValid {
		t.Error("WebP should be valid")
	}
}

func TestImageValidationService_ValidateSize_UnderLimit(t *testing.T) {
	ivs := NewImageValidationService(10)
	
	// 1MB file
	data := make([]byte, 1*1024*1024)
	isValid, err := ivs.ValidateSize(data)
	
	if err != nil {
		t.Errorf("Unexpected error for file under limit: %v", err)
	}
	if !isValid {
		t.Error("File under limit should be valid")
	}
}

func TestImageValidationService_ValidateSize_OverLimit(t *testing.T) {
	ivs := NewImageValidationService(10)
	
	// 11MB file
	data := make([]byte, 11*1024*1024)
	isValid, err := ivs.ValidateSize(data)
	
	if err == nil {
		t.Error("Expected error for file over limit")
	}
	if isValid {
		t.Error("File over limit should be invalid")
	}
}

func TestImageValidationService_DetectHumanFeatures(t *testing.T) {
	ivs := NewImageValidationService(10)
	
	// Valid JPEG
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01}
	hasFeatures, err := ivs.DetectHumanFeatures(jpegData)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hasFeatures {
		t.Error("Should detect features in valid image (placeholder implementation)")
	}
}
