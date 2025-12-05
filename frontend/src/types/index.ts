// Application types

export type ProcessingStatus =
  | "idle"
  | "uploading"
  | "processing"
  | "completed"
  | "failed";

export interface UploadedImage {
  file: File;
  fileId: string;
  previewUrl: string;
}

export interface ProcessingJob {
  jobId: string;
  status: "pending" | "processing" | "completed" | "failed";
  progress: number;
  message: string;
  estimatedTime?: number;
  resultUrl?: string;
}

export interface AppState {
  userPhoto: UploadedImage | null;
  shirtImage: UploadedImage | null;
  processingJob: ProcessingJob | null;
  error: string | null;
}
