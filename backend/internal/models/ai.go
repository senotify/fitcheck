package models

// AIRequestOptions contains optional parameters for AI service requests
type AIRequestOptions struct {
	Quality      string `json:"quality"`      // "standard" or "high"
	PreserveFace bool   `json:"preserveFace"`
}

// ReplicateInput represents the input for Replicate API (flux-vton)
type ReplicateInput struct {
	Image   string `json:"image"`   // Person image - base64 data URI or URL
	Garment string `json:"garment"` // Garment image - base64 data URI or URL
	Part    string `json:"part"`    // Body part: "upper", "lower", or "dress"
}

// AITryOnRequest represents a request to the AI service for virtual try-on (Replicate format)
type AITryOnRequest struct {
	Version string         `json:"version,omitempty"` // Optional - uses latest if not specified
	Input   ReplicateInput `json:"input"`
}

// AITryOnResponse represents a response from the AI service
type AITryOnResponse struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"` // "starting", "processing", "succeeded", "failed"
	Output interface{}            `json:"output,omitempty"` // Can be string URL or array of URLs
	Error  *string                `json:"error,omitempty"`
	URLs   *ReplicateResponseURLs `json:"urls,omitempty"`
}

// ReplicateResponseURLs contains URLs from Replicate response
type ReplicateResponseURLs struct {
	Get    string `json:"get"`
	Cancel string `json:"cancel"`
	Stream string `json:"stream,omitempty"`
}
