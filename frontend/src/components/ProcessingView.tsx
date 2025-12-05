import React, { useEffect, useState, useCallback } from "react";
import { apiClient } from "../api/client";
import { ProcessingJob } from "../types";
import "./ProcessingView.css";

interface ProcessingViewProps {
  jobId: string;
  onComplete: (resultUrl: string) => void;
  onError: (error: string) => void;
}

export const ProcessingView: React.FC<ProcessingViewProps> = ({
  jobId,
  onComplete,
  onError,
}) => {
  const [job, setJob] = useState<ProcessingJob>({
    jobId,
    status: "pending",
    progress: 0,
    message: "Starting processing...",
  });

  const pollStatus = useCallback(async () => {
    try {
      const statusResponse = await apiClient.getJobStatus(jobId);

      const updatedJob: ProcessingJob = {
        jobId,
        status: statusResponse.status,
        progress: statusResponse.progress,
        message: statusResponse.message,
        resultUrl: statusResponse.resultUrl,
      };

      setJob(updatedJob);

      // Handle completion
      if (statusResponse.status === "completed" && statusResponse.resultUrl) {
        onComplete(statusResponse.resultUrl);
      } else if (statusResponse.status === "failed") {
        onError(statusResponse.message || "Processing failed");
      }
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Failed to check status";
      onError(errorMessage);
    }
  }, [jobId, onComplete, onError]);

  // Poll status every 5 seconds
  useEffect(() => {
    // Initial poll
    pollStatus();

    // Set up polling interval
    const intervalId = setInterval(() => {
      // Only poll if still processing
      if (job.status === "pending" || job.status === "processing") {
        pollStatus();
      }
    }, 5000);

    // Cleanup interval on unmount
    return () => clearInterval(intervalId);
  }, [pollStatus, job.status]);

  // Format estimated time
  const formatEstimatedTime = (seconds?: number): string => {
    if (!seconds) return "Calculating...";
    if (seconds < 60) return `${seconds} seconds`;
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    if (remainingSeconds === 0)
      return `${minutes} minute${minutes > 1 ? "s" : ""}`;
    return `${minutes}m ${remainingSeconds}s`;
  };

  return (
    <div className="processing-view">
      <div className="processing-container">
        {/* Loading Animation */}
        <div className="loading-animation">
          <div className="spinner"></div>
          <div className="pulse-ring"></div>
        </div>

        {/* Status Message */}
        <h2 className="processing-title">Processing Your Virtual Try-On</h2>
        <p className="status-message">{job.message}</p>

        {/* Progress Bar */}
        <div className="progress-container">
          <div className="progress-bar-bg">
            <div
              className="progress-bar-fill"
              style={{ width: `${job.progress}%` }}
            >
              <span className="progress-text">{job.progress}%</span>
            </div>
          </div>
        </div>

        {/* Estimated Time */}
        {job.estimatedTime && (
          <div className="estimated-time">
            <svg
              className="clock-icon"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <circle cx="12" cy="12" r="10" />
              <polyline points="12 6 12 12 16 14" />
            </svg>
            <span>
              Estimated time: {formatEstimatedTime(job.estimatedTime)}
            </span>
          </div>
        )}

        {/* Status Indicator */}
        <div className="status-indicator">
          <div className={`status-dot ${job.status}`}></div>
          <span className="status-label">
            {job.status === "pending" && "Queued"}
            {job.status === "processing" && "Processing"}
            {job.status === "completed" && "Completed"}
            {job.status === "failed" && "Failed"}
          </span>
        </div>
      </div>
    </div>
  );
};
