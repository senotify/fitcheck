import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { apiClient } from "../api";
import "./JobDetailView.css";

interface Job {
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

export const JobDetailView: React.FC = () => {
  const { jobId } = useParams<{ jobId: string }>();
  const navigate = useNavigate();
  const [job, setJob] = useState<Job | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchJob = async () => {
      if (!jobId) return;

      try {
        setLoading(true);
        // Fetch from jobs list to get full details including images
        const response = await apiClient.getJobs({});
        console.log("Jobs response:", response);
        const foundJob = response.jobs?.find((j: any) => j.jobId === jobId);
        console.log("Found job:", foundJob);

        if (foundJob) {
          setJob(foundJob);
        } else {
          setError("Job not found");
        }
      } catch (err) {
        setError("Failed to load job details");
        console.error("Error fetching job:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchJob();
  }, [jobId]);

  const getFullUrl = (url?: string) => {
    // Backend now sends full URLs, just return as-is
    return url;
  };

  const handleDownload = () => {
    if (job?.resultUrl) {
      const link = document.createElement("a");
      link.href = getFullUrl(job.resultUrl) || "";
      link.download = `virtual-fitcheck-${jobId}.jpg`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }
  };

  if (loading) {
    return (
      <div className="job-detail-view">
        <div className="loading-container">
          <div className="spinner"></div>
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  if (error || !job) {
    return (
      <div className="job-detail-view">
        <div className="error-container">
          <h2>Error</h2>
          <p>{error || "Job not found"}</p>
          <button onClick={() => navigate("/jobs")} className="back-button">
            Back to My Fits
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="job-detail-view">
      <div className="job-detail-header">
        <button onClick={() => navigate("/fits")} className="back-button">
          ← Back to My Fits
        </button>
        <h1>Fit Result</h1>
      </div>

      <div className="job-detail-content">
        {job.status === "completed" && job.resultUrl ? (
          <div className="result-container">
            {/* Main Result Display - Now on Top */}
            <div className="main-result">
              <div className="result-header">
                <h2>Your Virtual Try-On</h2>
                <div className="result-badge">
                  <svg
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                  >
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                  Completed
                </div>
              </div>
              <div className="result-image-wrapper">
                <img
                  src={getFullUrl(job.resultUrl)}
                  alt="Virtual try-on result"
                  className="result-image"
                />
              </div>
              <div className="result-actions">
                <button onClick={handleDownload} className="download-button">
                  <svg
                    width="20"
                    height="20"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                  >
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="7 10 12 15 17 10" />
                    <line x1="12" y1="15" x2="12" y2="3" />
                  </svg>
                  Download Image
                </button>
                <button
                  onClick={() => navigate("/fits")}
                  className="secondary-button"
                >
                  View More Fits
                </button>
              </div>
            </div>

            {/* Before/After Comparison - Now Below */}
            <div className="comparison-section">
              <h3 className="section-title">Source Images</h3>
              <div className="comparison-grid">
                <div className="comparison-item">
                  <div className="comparison-label">Original Photo</div>
                  <div className="comparison-image-wrapper">
                    <img
                      src={getFullUrl(job.userPhotoUrl)}
                      alt="Original photo"
                      className="comparison-image"
                    />
                  </div>
                </div>

                <div className="comparison-item">
                  <div className="comparison-label">Shirt Design</div>
                  <div className="comparison-image-wrapper">
                    <img
                      src={getFullUrl(job.shirtImageUrl)}
                      alt="Shirt"
                      className="comparison-image"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        ) : job.status === "processing" || job.status === "pending" ? (
          <div className="processing-container">
            <div className="spinner-large"></div>
            <h2>Processing Your Fit</h2>
            <p>{job.statusMessage || "Please wait..."}</p>
            <div className="progress-bar">
              <div
                className="progress-fill"
                style={{ width: `${job.progress}%` }}
              />
            </div>
            <span className="progress-text">{job.progress}%</span>
          </div>
        ) : (
          <div className="error-container">
            <h2>Processing Failed</h2>
            <p>{job.error || job.statusMessage || "An error occurred"}</p>
            <button onClick={() => navigate("/jobs")} className="back-button">
              Back to My Fits
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
