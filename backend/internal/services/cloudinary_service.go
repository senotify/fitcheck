package services

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// CloudinaryService handles image uploads to Cloudinary
type CloudinaryService struct {
	cloudName string
	apiKey    string
	apiSecret string
	client    *http.Client
}

// NewCloudinaryService creates a new Cloudinary service
func NewCloudinaryService(cloudName, apiKey, apiSecret string) *CloudinaryService {
	return &CloudinaryService{
		cloudName: cloudName,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CloudinaryResponse represents the response from Cloudinary
type CloudinaryResponse struct {
	SecureURL string `json:"secure_url"`
	PublicID  string `json:"public_id"`
	URL       string `json:"url"`
}

// UploadImage uploads an image to Cloudinary and returns the public URL
func (s *CloudinaryService) UploadImage(imageData []byte, filename string) (string, error) {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := part.Write(imageData); err != nil {
		return "", fmt.Errorf("failed to write image data: %w", err)
	}

	// Add API key for authenticated upload
	writer.WriteField("api_key", s.apiKey)
	
	// Add timestamp
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	writer.WriteField("timestamp", timestamp)
	
	// Generate signature for authenticated upload
	// Signature = SHA1(timestamp=<timestamp>&api_secret=<api_secret>)
	signature := s.generateSignature(timestamp)
	writer.WriteField("signature", signature)

	// Close the writer
	writer.Close()

	// Create upload URL
	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", s.cloudName)

	// Create request
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload to Cloudinary: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Cloudinary returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var cloudinaryResp CloudinaryResponse
	if err := json.Unmarshal(respBody, &cloudinaryResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Return secure URL
	return cloudinaryResp.SecureURL, nil
}

// generateSignature creates a SHA1 signature for Cloudinary authenticated upload
func (s *CloudinaryService) generateSignature(timestamp string) string {
	// Cloudinary signature format: SHA1(timestamp=<timestamp>&api_secret=<api_secret>)
	signatureString := fmt.Sprintf("timestamp=%s%s", timestamp, s.apiSecret)
	hash := sha1.New()
	hash.Write([]byte(signatureString))
	return hex.EncodeToString(hash.Sum(nil))
}
