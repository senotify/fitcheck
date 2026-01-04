import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { apiClient } from "../api";
import { JobCard, Job } from "./JobCard";
import "./JobHistoryView.css";

export const JobHistoryView: React.FC = () => {
  const navigate = useNavigate();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>("all");
  const [sortOrder, setSortOrder] = useState<string>("newest");
  const [showToast, setShowToast] = useState(false);
  const [toastMessage, setToastMessage] = useState("");
  const [toastType, setToastType] = useState<"success" | "error">("success");

  const fetchJobs = async () => {
    try {
      setLoading(true);
      const params: any = { sort: sortOrder };
      if (filter !== "all") {
        params.status = filter;
      }

      const response = await apiClient.getJobs(params);
      if (response.success) {
        setJobs(response.jobs || []);
      }
    } catch (error) {
      console.error("Failed to fetch jobs:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchJobs();
    // Poll for updates every 5 seconds
    const interval = setInterval(fetchJobs, 5000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter, sortOrder]);

  const showToastNotification = (
    message: string,
    type: "success" | "error" = "success"
  ) => {
    setToastMessage(message);
    setToastType(type);
    setShowToast(true);
    setTimeout(() => setShowToast(false), 3000);
  };

  const handleDelete = async (jobId: string) => {
    // The JobCard component already handles the confirmation modal
    try {
      await apiClient.deleteJob(jobId);
      // Refresh the list
      fetchJobs();
      // Show success toast
      showToastNotification("Fit deleted successfully!");
    } catch (error) {
      console.error("Failed to delete job:", error);
      showToastNotification("Failed to delete fit. Please try again.", "error");
    }
  };

  const handleView = (jobId: string) => {
    // Navigate to the detail page
    navigate(`/fits/${jobId}`);
  };

  // No need to fix URLs anymore - backend sends full URLs

  return (
    <div className="job-history-view">
      {/* Toast Notification */}
      {showToast && (
        <div className={`toast-notification ${toastType}`}>
          <div className="toast-content">
            {toastType === "success" ? (
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
            ) : (
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="15" y1="9" x2="9" y2="15" />
                <line x1="9" y1="9" x2="15" y2="15" />
              </svg>
            )}
            <span>{toastMessage}</span>
          </div>
        </div>
      )}

      <div className="job-history-header">
        <h2>My Fits</h2>
        <p>View and manage your virtual try-on history</p>
      </div>

      <div className="job-history-controls">
        <div className="filter-controls">
          <label>Filter:</label>
          <select value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="all">All</option>
            <option value="pending">Pending</option>
            <option value="processing">Processing</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
        </div>

        <div className="sort-controls">
          <label>Sort:</label>
          <select
            value={sortOrder}
            onChange={(e) => setSortOrder(e.target.value)}
          >
            <option value="newest">Newest First</option>
            <option value="oldest">Oldest First</option>
          </select>
        </div>
      </div>

      <div className="job-history-content">
        {loading && jobs.length === 0 ? (
          <p>Loading your fits...</p>
        ) : jobs.length === 0 ? (
          <p>No fits found. Start by creating your first virtual try-on!</p>
        ) : (
          <div className="job-list">
            {jobs.map((job) => (
              <JobCard
                key={job.jobId}
                job={job}
                onDelete={handleDelete}
                onView={handleView}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
