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

  const [startTime] = useState<number>(Date.now());
  const [elapsedTime, setElapsedTime] = useState<number>(0);
  const [showLongProcessingModal, setShowLongProcessingModal] =
    useState<boolean>(false);
  const [hasShownNotification, setHasShownNotification] =
    useState<boolean>(false);
  const [email, setEmail] = useState<string>("");
  const [emailError, setEmailError] = useState<string>("");
  const [emailSubmitted, setEmailSubmitted] = useState<boolean>(false);

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

  // Show email notification popup immediately on mount
  useEffect(() => {
    // Show the notification popup immediately when processing starts
    if (!hasShownNotification && !emailSubmitted) {
      setShowLongProcessingModal(true);
      setHasShownNotification(true);
    }
  }, []); // Run once on mount

  // Track elapsed time
  useEffect(() => {
    // Don't track time if job is completed or failed
    if (job.status === "completed" || job.status === "failed") {
      return;
    }

    const timeIntervalId = setInterval(() => {
      const elapsed = Math.floor((Date.now() - startTime) / 1000);
      setElapsedTime(elapsed);
    }, 1000);

    return () => clearInterval(timeIntervalId);
  }, [startTime, job.status]);

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

  // Format elapsed time
  const formatElapsedTime = (seconds: number): string => {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}:${remainingSeconds.toString().padStart(2, "0")}`;
  };

  // Validate email
  const validateEmail = (email: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  };

  // Handle email submission
  const handleEmailSubmit = async () => {
    setEmailError("");

    if (!email.trim()) {
      setEmailError("Please enter an email address");
      return;
    }

    if (!validateEmail(email)) {
      setEmailError("Please enter a valid email address");
      return;
    }

    try {
      await apiClient.notifyEmail(jobId, email);
      setEmailSubmitted(true);
      setShowLongProcessingModal(false);
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Failed to register email";
      setEmailError(errorMessage);
    }
  };

  // Handle continue waiting
  const handleContinueWaiting = () => {
    setShowLongProcessingModal(false);
  };

  return (
    <div className="processing-view">
      <div className="processing-container">
        {/* Status Message */}
        <h2 className="processing-title">Processing Your Virtual Try-On</h2>
        <p className="status-message">{job.message}</p>

        {/* Loading Animation */}
        <div className="loading-animation">
          <div className="spinner"></div>
        </div>

        {/* Progress Bar */}
        <div className="progress-container">
          <div className="progress-bar-bg">
            <div
              className="progress-bar-fill"
              style={{ width: `${job.progress}%` }}
            />
          </div>
          <span className="progress-text">{job.progress}%</span>
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

        {/* Elapsed Time */}
        <div className="elapsed-time">
          <span>Elapsed: {formatElapsedTime(elapsedTime)}</span>
        </div>

        {/* Email Submitted Confirmation */}
        {emailSubmitted && (
          <div className="email-confirmation">
            <p>✓ We'll email you at {email} when processing completes!</p>
            <p className="email-note">You can close this page safely.</p>
          </div>
        )}
      </div>

      {/* Email Notification Modal */}
      {showLongProcessingModal && (
        <div className="modal-overlay">
          <div className="modal-content">
            <h3>Get notified when your result is ready</h3>
            <p>
              Processing may take a few minutes. Enter your email to receive a
              notification when your virtual try-on is complete, or continue
              watching the progress here.
            </p>

            <div className="modal-options">
              <div className="email-option">
                <h4>Get notified by email</h4>
                <input
                  type="email"
                  placeholder="Enter your email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className={emailError ? "error" : ""}
                />
                {emailError && <p className="error-message">{emailError}</p>}
                <button
                  className="btn-primary"
                  onClick={handleEmailSubmit}
                  disabled={!email.trim()}
                >
                  Send me an email
                </button>
              </div>

              <div className="divider">
                <span>OR</span>
              </div>

              <div className="continue-option">
                <h4>Keep waiting</h4>
                <p>Continue watching the progress on this page</p>
                <button
                  className="btn-secondary"
                  onClick={handleContinueWaiting}
                >
                  Continue waiting
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
