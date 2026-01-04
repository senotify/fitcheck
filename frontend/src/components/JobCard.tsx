import React, { useState } from "react";
import "./JobCard.css";

export interface Job {
  jobId: string;
  status: "pending" | "processing" | "completed" | "failed";
  progress: number;
  statusMessage?: string;
  resultUrl?: string;
  thumbnailUrl?: string;
  userPhotoUrl?: string;
  shirtImageUrl?: string;
  createdAt: string;
  completedAt?: string;
  error?: string;
}

interface JobCardProps {
  job: Job;
  onDelete: (jobId: string) => void;
  onView: (jobId: string) => void;
}

export const JobCard: React.FC<JobCardProps> = ({ job, onDelete, onView }) => {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Format date to readable string
  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60)
      return `${diffMins} minute${diffMins > 1 ? "s" : ""} ago`;
    if (diffHours < 24)
      return `${diffHours} hour${diffHours > 1 ? "s" : ""} ago`;
    if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? "s" : ""} ago`;

    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: date.getFullYear() !== now.getFullYear() ? "numeric" : undefined,
    });
  };

  // Get status display info
  const getStatusInfo = () => {
    switch (job.status) {
      case "pending":
        return { label: "Queued", color: "#f39c12" };
      case "processing":
        return { label: "Processing", color: "#5dade2" };
      case "completed":
        return { label: "Completed", color: "#2ecc71" };
      case "failed":
        return { label: "Failed", color: "#ff6b6b" };
      default:
        return { label: "Unknown", color: "#9ba3af" };
    }
  };

  const statusInfo = getStatusInfo();

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent card click
    setShowDeleteConfirm(true);
  };

  const handleConfirmDelete = () => {
    onDelete(job.jobId);
    setShowDeleteConfirm(false);
  };

  const handleCancelDelete = () => {
    setShowDeleteConfirm(false);
  };

  const handleViewResult = (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent card click
    onView(job.jobId);
  };

  const handleCardClick = () => {
    // Only navigate if job is completed
    if (job.status === "completed" && job.resultUrl) {
      onView(job.jobId);
    }
  };

  return (
    <>
      <div
        className={`job-card ${
          job.status === "completed" && job.resultUrl ? "clickable" : ""
        }`}
        onClick={handleCardClick}
      >
        {/* Thumbnail */}
        <div className="job-thumbnail">
          {job.thumbnailUrl ? (
            <img src={job.thumbnailUrl} alt="Job preview" />
          ) : (
            <div className="thumbnail-placeholder">
              <svg
                width="40"
                height="40"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
            </div>
          )}
        </div>

        {/* Job Info */}
        <div className="job-info">
          <div className="job-header">
            <div className="job-status">
              <div
                className="status-dot"
                style={{ backgroundColor: statusInfo.color }}
              />
              <span className="status-label">{statusInfo.label}</span>
            </div>
            <span className="job-time">{formatDate(job.createdAt)}</span>
          </div>

          {/* Progress bar for in-progress jobs */}
          {(job.status === "pending" || job.status === "processing") && (
            <div className="job-progress">
              <div className="progress-bar-bg">
                <div
                  className="progress-bar-fill"
                  style={{ width: `${job.progress}%` }}
                />
              </div>
              <span className="progress-text">{job.progress}%</span>
            </div>
          )}

          {/* Status message or error */}
          {job.statusMessage && job.status !== "failed" && (
            <p className="job-message">{job.statusMessage}</p>
          )}
          {job.error && job.status === "failed" && (
            <p className="job-error">{job.error}</p>
          )}
        </div>

        {/* Actions */}
        <div className="job-actions">
          {job.status === "completed" && job.resultUrl && (
            <button
              className="view-button"
              onClick={handleViewResult}
              aria-label="View result"
            >
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
              View
            </button>
          )}
          <button
            className="delete-button"
            onClick={handleDeleteClick}
            aria-label="Delete job"
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
            </svg>
            Delete
          </button>
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="modal-overlay" onClick={handleCancelDelete}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <h3>Delete Job?</h3>
            <p>
              Are you sure you want to delete this job? This action cannot be
              undone.
            </p>
            <div className="modal-actions">
              <button className="btn-cancel" onClick={handleCancelDelete}>
                Cancel
              </button>
              <button className="btn-delete" onClick={handleConfirmDelete}>
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
};
