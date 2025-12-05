import React, {
  useState,
  useRef,
  useCallback,
  DragEvent,
  ChangeEvent,
} from "react";
import "./UploadInterface.css";

interface UploadInterfaceProps {
  onUpload: (file: File) => Promise<void>;
  uploadProgress: number;
  previewUrl: string | null;
  label: string;
  accept?: string;
  error?: string | null;
}

const ACCEPTED_FORMATS = ["image/jpeg", "image/png", "image/webp"];
const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB in bytes

export const UploadInterface: React.FC<UploadInterfaceProps> = ({
  onUpload,
  uploadProgress,
  previewUrl,
  label,
  accept = "image/jpeg,image/png,image/webp",
  error: externalError,
}) => {
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [uploadSuccess, setUploadSuccess] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Client-side validation
  const validateFile = (file: File): { valid: boolean; error?: string } => {
    // Validate file type
    if (!ACCEPTED_FORMATS.includes(file.type)) {
      return {
        valid: false,
        error: `Invalid file format. Please upload JPEG, PNG, or WebP images.`,
      };
    }

    // Validate file size
    if (file.size > MAX_FILE_SIZE) {
      return {
        valid: false,
        error: `File size exceeds 10MB limit. Please upload a smaller image.`,
      };
    }

    return { valid: true };
  };

  // Handle file upload
  const handleFileUpload = useCallback(
    async (file: File) => {
      // Reset states
      setValidationError(null);
      setUploadSuccess(false);

      // Validate file
      const validation = validateFile(file);
      if (!validation.valid) {
        setValidationError(validation.error || "Invalid file");
        return;
      }

      // Upload file
      setIsUploading(true);
      try {
        await onUpload(file);
        setUploadSuccess(true);
        // Clear success message after 3 seconds
        setTimeout(() => setUploadSuccess(false), 3000);
      } catch (error) {
        // Error is handled by parent component
        console.error("Upload failed:", error);
      } finally {
        setIsUploading(false);
      }
    },
    [onUpload]
  );

  // Handle drag events
  const handleDragEnter = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleDragOver = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback(
    (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(false);

      const files = e.dataTransfer.files;
      if (files && files.length > 0) {
        handleFileUpload(files[0]);
      }
    },
    [handleFileUpload]
  );

  // Handle file input change
  const handleFileInputChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const files = e.target.files;
      if (files && files.length > 0) {
        handleFileUpload(files[0]);
      }
    },
    [handleFileUpload]
  );

  // Handle click to open file dialog
  const handleClick = useCallback(() => {
    if (!isUploading && fileInputRef.current) {
      fileInputRef.current.click();
    }
  }, [isUploading]);

  // Determine which error to display
  const displayError = validationError || externalError;

  return (
    <div className="upload-interface">
      <label className="upload-label">{label}</label>

      <div
        className={`upload-dropzone ${isDragging ? "dragging" : ""} ${
          previewUrl ? "has-preview" : ""
        } ${isUploading ? "uploading" : ""}`}
        onDragEnter={handleDragEnter}
        onDragLeave={handleDragLeave}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
        onClick={handleClick}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept={accept}
          onChange={handleFileInputChange}
          style={{ display: "none" }}
          disabled={isUploading}
        />

        {previewUrl ? (
          <div className="preview-container">
            <img src={previewUrl} alt="Preview" className="preview-image" />
            {!isUploading && (
              <div className="preview-overlay">
                <span>Click or drag to replace</span>
              </div>
            )}
          </div>
        ) : (
          <div className="upload-prompt">
            <svg
              className="upload-icon"
              width="48"
              height="48"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="17 8 12 3 7 8" />
              <line x1="12" y1="3" x2="12" y2="15" />
            </svg>
            <p className="upload-text">
              <strong>Click to upload</strong> or drag and drop
            </p>
            <p className="upload-hint">JPEG, PNG, or WebP (max 10MB)</p>
          </div>
        )}

        {isUploading && (
          <div className="upload-progress-overlay">
            <div className="progress-bar-container">
              <div
                className="progress-bar"
                style={{ width: `${uploadProgress}%` }}
              />
            </div>
            <p className="progress-text">{uploadProgress}%</p>
          </div>
        )}
      </div>

      {displayError && (
        <div className="error-message" role="alert">
          {displayError}
        </div>
      )}

      {uploadSuccess && !displayError && (
        <div className="success-message" role="status">
          Upload successful!
        </div>
      )}
    </div>
  );
};
