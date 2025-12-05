// API client for backend communication

const API_BASE_URL = process.env.REACT_APP_API_URL || "http://localhost:8080";

export interface UploadResponse {
  success: boolean;
  fileId: string;
  previewUrl: string;
}

export interface ProcessResponse {
  success: boolean;
  jobId: string;
  estimatedTime: number;
}

export interface StatusResponse {
  status: "pending" | "processing" | "completed" | "failed";
  progress: number;
  message: string;
  resultUrl?: string;
}

export interface ErrorResponse {
  success: false;
  error: {
    code: string;
    message: string;
    details?: any;
  };
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
  }

  /**
   * Upload user photo
   */
  async uploadUserPhoto(file: File): Promise<UploadResponse> {
    const formData = new FormData();
    formData.append("image", file);

    const response = await fetch(`${this.baseUrl}/api/upload/user-photo`, {
      method: "POST",
      body: formData,
    });

    if (!response.ok) {
      const error: ErrorResponse = await response.json();
      throw new Error(error.error.message || "Failed to upload user photo");
    }

    return response.json();
  }

  /**
   * Upload shirt image
   */
  async uploadShirt(file: File): Promise<UploadResponse> {
    const formData = new FormData();
    formData.append("image", file);

    const response = await fetch(`${this.baseUrl}/api/upload/shirt`, {
      method: "POST",
      body: formData,
    });

    if (!response.ok) {
      const error: ErrorResponse = await response.json();
      throw new Error(error.error.message || "Failed to upload shirt image");
    }

    return response.json();
  }

  /**
   * Initiate processing
   */
  async processImages(
    userPhotoId: string,
    shirtImageId: string
  ): Promise<ProcessResponse> {
    const response = await fetch(`${this.baseUrl}/api/process`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        userPhotoId,
        shirtImageId,
      }),
    });

    if (!response.ok) {
      const error: ErrorResponse = await response.json();
      throw new Error(error.error.message || "Failed to process images");
    }

    return response.json();
  }

  /**
   * Check job status
   */
  async getJobStatus(jobId: string): Promise<StatusResponse> {
    const response = await fetch(`${this.baseUrl}/api/status/${jobId}`, {
      method: "GET",
    });

    if (!response.ok) {
      const error: ErrorResponse = await response.json();
      throw new Error(error.error.message || "Failed to get job status");
    }

    return response.json();
  }

  /**
   * Get preview URL for uploaded file
   */
  getPreviewUrl(fileId: string): string {
    return `${this.baseUrl}/api/preview/${fileId}`;
  }

  /**
   * Get result download URL
   */
  getResultUrl(resultId: string): string {
    return `${this.baseUrl}/api/result/${resultId}`;
  }
}

// Export singleton instance
export const apiClient = new ApiClient();
