// API client for backend communication

const API_BASE_URL = process.env.REACT_APP_API_URL || "";

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

export interface EmailNotificationRequest {
  jobId: string;
  email: string;
}

export interface EmailNotificationResponse {
  success: boolean;
  message: string;
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
      credentials: "include", // Include cookies for session
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
      credentials: "include", // Include cookies for session
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
      credentials: "include", // Include cookies for session
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
      credentials: "include", // Include cookies for session
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

  /**
   * Register email notification for a job
   */
  async notifyEmail(
    jobId: string,
    email: string
  ): Promise<EmailNotificationResponse> {
    const response = await fetch(`${this.baseUrl}/api/notify-email`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include", // Include cookies for session
      body: JSON.stringify({
        jobId,
        email,
      }),
    });

    if (!response.ok) {
      const error: ErrorResponse = await response.json();
      throw new Error(
        error.error.message || "Failed to register email notification"
      );
    }

    return response.json();
  }

  /**
   * Get list of jobs with optional filters
   */
  async getJobs(params?: {
    status?: string;
    sort?: string;
    limit?: number;
    offset?: number;
  }): Promise<any> {
    const queryParams = new URLSearchParams();
    if (params?.status) queryParams.append("status", params.status);
    if (params?.sort) queryParams.append("sort", params.sort);
    if (params?.limit) queryParams.append("limit", params.limit.toString());
    if (params?.offset) queryParams.append("offset", params.offset.toString());

    const url = `${this.baseUrl}/api/jobs${
      queryParams.toString() ? `?${queryParams.toString()}` : ""
    }`;

    const response = await fetch(url, {
      method: "GET",
      credentials: "include", // Include cookies for session
    });

    if (!response.ok) {
      const error: ErrorResponse = await response.json();
      throw new Error(error.error.message || "Failed to get jobs");
    }

    return response.json();
  }

  /**
   * Delete a job
   */
  async deleteJob(jobId: string): Promise<any> {
    const response = await fetch(`${this.baseUrl}/api/jobs/${jobId}`, {
      method: "DELETE",
      credentials: "include", // Include cookies for session
    });

    if (!response.ok) {
      const error: ErrorResponse = await response.json();
      throw new Error(error.error.message || "Failed to delete job");
    }

    return response.json();
  }
}

// Export singleton instance
export const apiClient = new ApiClient();
