import React, { useState, useCallback } from "react";
import { BrowserRouter as Router, Routes, Route, Link } from "react-router-dom";
import "./App.css";
import { AppState, UploadedImage, ProcessingJob } from "./types";
import { apiClient } from "./api/client";
import { LandingPage } from "./components/LandingPage";
import { UploadInterface } from "./components/UploadInterface";
import { ProcessingView } from "./components/ProcessingView";
import { ResultDisplay } from "./components/ResultDisplay";
import { JobHistoryView } from "./components/JobHistoryView";
import { JobDetailView } from "./components/JobDetailView";

function App() {
  const [showLanding, setShowLanding] = useState(true);
  const [state, setState] = useState<AppState>({
    userPhoto: null,
    shirtImage: null,
    processingJob: null,
    error: null,
  });

  const [userPhotoProgress, setUserPhotoProgress] = useState(0);
  const [shirtProgress, setShirtProgress] = useState(0);
  const [userPhotoError, setUserPhotoError] = useState<string | null>(null);
  const [shirtError, setShirtError] = useState<string | null>(null);

  // Upload user photo handler
  const handleUserPhotoUpload = useCallback(async (file: File) => {
    try {
      setUserPhotoError(null);
      setUserPhotoProgress(10); // Start at 10% to show immediate feedback
      setState((prev) => ({ ...prev, error: null }));

      // Ensure minimum display time for progress bar
      const minDisplayTime = 800; // Minimum 800ms to show progress
      const startTime = Date.now();

      // Simulate progress (in real implementation, use XMLHttpRequest for progress tracking)
      const progressInterval = setInterval(() => {
        setUserPhotoProgress((prev) => Math.min(prev + 10, 90));
      }, 80);

      const response = await apiClient.uploadUserPhoto(file);

      clearInterval(progressInterval);
      setUserPhotoProgress(100);

      // Ensure progress bar is visible for minimum time
      const elapsed = Date.now() - startTime;
      const remainingTime = Math.max(0, minDisplayTime - elapsed);

      await new Promise((resolve) => setTimeout(resolve, remainingTime));

      const uploadedImage: UploadedImage = {
        file,
        fileId: response.fileId,
        previewUrl: response.previewUrl.startsWith("http") ? response.previewUrl : `http://localhost:8080${response.previewUrl}`, // Use previewUrl from backend response
      };
      setState((prev) => ({ ...prev, userPhoto: uploadedImage }));

      // Reset progress after showing completion
      setTimeout(() => setUserPhotoProgress(0), 1500);
    } catch (error) {
      setUserPhotoProgress(0);
      const errorMessage =
        error instanceof Error ? error.message : "Failed to upload user photo";
      setUserPhotoError(errorMessage);
      setState((prev) => ({
        ...prev,
        error: errorMessage,
      }));
    }
  }, []);

  // Upload shirt image handler
  const handleShirtUpload = useCallback(async (file: File) => {
    try {
      setShirtError(null);
      setShirtProgress(10); // Start at 10% to show immediate feedback
      setState((prev) => ({ ...prev, error: null }));

      // Ensure minimum display time for progress bar
      const minDisplayTime = 800; // Minimum 800ms to show progress
      const startTime = Date.now();

      // Simulate progress (in real implementation, use XMLHttpRequest for progress tracking)
      const progressInterval = setInterval(() => {
        setShirtProgress((prev) => Math.min(prev + 10, 90));
      }, 80);

      const response = await apiClient.uploadShirt(file);

      clearInterval(progressInterval);
      setShirtProgress(100);

      // Ensure progress bar is visible for minimum time
      const elapsed = Date.now() - startTime;
      const remainingTime = Math.max(0, minDisplayTime - elapsed);

      await new Promise((resolve) => setTimeout(resolve, remainingTime));

      const uploadedImage: UploadedImage = {
        file,
        fileId: response.fileId,
        previewUrl: response.previewUrl.startsWith("http") ? response.previewUrl : `http://localhost:8080${response.previewUrl}`, // Use previewUrl from backend response
      };
      setState((prev) => ({ ...prev, shirtImage: uploadedImage }));

      // Reset progress after showing completion
      setTimeout(() => setShirtProgress(0), 1500);
    } catch (error) {
      setShirtProgress(0);
      const errorMessage =
        error instanceof Error ? error.message : "Failed to upload shirt image";
      setShirtError(errorMessage);
      setState((prev) => ({
        ...prev,
        error: errorMessage,
      }));
    }
  }, []);

  // Process images handler
  const handleProcess = useCallback(async () => {
    if (!state.userPhoto || !state.shirtImage) {
      setState((prev) => ({
        ...prev,
        error: "Please upload both user photo and shirt image",
      }));
      return;
    }

    try {
      setState((prev) => ({ ...prev, error: null }));
      const response = await apiClient.processImages(
        state.userPhoto.fileId,
        state.shirtImage.fileId
      );

      const job: ProcessingJob = {
        jobId: response.jobId,
        status: "pending",
        progress: 0,
        message: "Starting processing...",
        estimatedTime: response.estimatedTime,
      };

      setState((prev) => ({ ...prev, processingJob: job }));
    } catch (error) {
      setState((prev) => ({
        ...prev,
        error:
          error instanceof Error ? error.message : "Failed to process images",
      }));
    }
  }, [state.userPhoto, state.shirtImage]);

  // Handle processing completion
  const handleProcessingComplete = useCallback((resultUrl: string) => {
    // Convert relative URL to absolute URL with backend server
    const fullResultUrl = resultUrl.startsWith("http")
      ? resultUrl
      : `${
          process.env.REACT_APP_API_URL || "http://localhost:8080"
        }${resultUrl}`;

    setState((prev) => ({
      ...prev,
      processingJob: prev.processingJob
        ? {
            ...prev.processingJob,
            status: "completed",
            resultUrl: fullResultUrl,
          }
        : null,
    }));
  }, []);

  // Handle processing error
  const handleProcessingError = useCallback((error: string) => {
    setState((prev) => ({
      ...prev,
      error,
      processingJob: prev.processingJob
        ? { ...prev.processingJob, status: "failed" }
        : null,
    }));
  }, []);

  // Handle download
  const handleDownload = useCallback(async () => {
    if (!state.processingJob?.resultUrl) return;

    try {
      // Fetch the image as a blob
      const response = await fetch(state.processingJob.resultUrl);
      const blob = await response.blob();

      // Create object URL and trigger download
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `virtual-fitcheck-${Date.now()}.jpg`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);

      // Clean up object URL
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error("Download failed:", error);
      setState((prev) => ({
        ...prev,
        error: "Failed to download image",
      }));
    }
  }, [state.processingJob?.resultUrl]);

  // Handle try another shirt
  const handleTryAnother = useCallback(() => {
    // Keep user photo, clear shirt and processing job
    setState((prev) => ({
      ...prev,
      shirtImage: null,
      processingJob: null,
      error: null,
    }));
    setShirtError(null);
  }, []);

  // Handle retry after error
  const handleRetry = useCallback(() => {
    setState((prev) => ({
      ...prev,
      processingJob: null,
      error: null,
    }));
  }, []);

  // Handle get started from landing page
  const handleGetStarted = useCallback(() => {
    setShowLanding(false);
  }, []);

  // Upload page component
  const UploadPage = () => (
    <div className="upload-container">
      <div className="upload-section">
        <UploadInterface
          label="Upload Your Photo"
          onUpload={handleUserPhotoUpload}
          uploadProgress={userPhotoProgress}
          previewUrl={state.userPhoto?.previewUrl || null}
          error={userPhotoError}
        />
      </div>

      <div className="upload-section">
        <UploadInterface
          label="Upload Shirt Image"
          onUpload={handleShirtUpload}
          uploadProgress={shirtProgress}
          previewUrl={state.shirtImage?.previewUrl || null}
          error={shirtError}
        />
      </div>

      {state.error && (
        <div className="global-error-message">
          {state.error}
          <button className="retry-button" onClick={handleRetry}>
            Try Again
          </button>
        </div>
      )}

      <div className="action-section">
        <button
          className="process-button"
          onClick={handleProcess}
          disabled={!state.userPhoto || !state.shirtImage}
        >
          Process Images
        </button>
      </div>
    </div>
  );

  return (
    <Router>
      <div className="App">
        <header className="App-header">
          <div className="header-content">
            <Link to="/" className="logo-link">
              <div className="logo">
                <svg
                  className="logo-icon"
                  width="32"
                  height="32"
                  viewBox="0 0 32 32"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <rect
                    x="8"
                    y="4"
                    width="16"
                    height="24"
                    rx="2"
                    stroke="url(#gradient1)"
                    strokeWidth="2"
                    fill="none"
                  />
                  <path
                    d="M12 10 L16 14 L20 10"
                    stroke="url(#gradient2)"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                  <circle cx="16" cy="20" r="2" fill="url(#gradient3)" />
                  <defs>
                    <linearGradient
                      id="gradient1"
                      x1="8"
                      y1="4"
                      x2="24"
                      y2="28"
                      gradientUnits="userSpaceOnUse"
                    >
                      <stop offset="0%" stopColor="#667eea" />
                      <stop offset="100%" stopColor="#764ba2" />
                    </linearGradient>
                    <linearGradient
                      id="gradient2"
                      x1="12"
                      y1="10"
                      x2="20"
                      y2="14"
                      gradientUnits="userSpaceOnUse"
                    >
                      <stop offset="0%" stopColor="#667eea" />
                      <stop offset="100%" stopColor="#764ba2" />
                    </linearGradient>
                    <linearGradient
                      id="gradient3"
                      x1="14"
                      y1="18"
                      x2="18"
                      y2="22"
                      gradientUnits="userSpaceOnUse"
                    >
                      <stop offset="0%" stopColor="#667eea" />
                      <stop offset="100%" stopColor="#764ba2" />
                    </linearGradient>
                  </defs>
                </svg>
                <div className="logo-text">
                  <h1>Virtual FitCheck</h1>
                  <p>AI-Powered Virtual Try-On</p>
                </div>
              </div>
            </Link>
            <nav className="header-nav">
              <Link to="/upload" className="nav-link">
                Upload
              </Link>
              <Link to="/fits" className="nav-link">
                My Fits
              </Link>
            </nav>
          </div>
        </header>
        <main className="App-main">
          <Routes>
            <Route
              path="/"
              element={
                showLanding ? (
                  <LandingPage onGetStarted={handleGetStarted} />
                ) : (
                  <UploadPage />
                )
              }
            />
            <Route
              path="/upload"
              element={
                <>
                  {/* Show upload interface when not processing and no result */}
                  {!state.processingJob && <UploadPage />}

                  {/* Show processing view when job is in progress */}
                  {state.processingJob &&
                    state.processingJob.status !== "completed" && (
                      <ProcessingView
                        jobId={state.processingJob.jobId}
                        onComplete={handleProcessingComplete}
                        onError={handleProcessingError}
                      />
                    )}

                  {/* Show result display when processing is complete */}
                  {state.processingJob &&
                    state.processingJob.status === "completed" &&
                    state.processingJob.resultUrl && (
                      <ResultDisplay
                        resultUrl={state.processingJob.resultUrl}
                        onDownload={handleDownload}
                        onTryAnother={handleTryAnother}
                      />
                    )}
                </>
              }
            />
            <Route path="/fits" element={<JobHistoryView />} />
            <Route path="/fits/:jobId" element={<JobDetailView />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;
