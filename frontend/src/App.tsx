import React, { useState, useCallback } from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import "./App.css";
import { AppState, UploadedImage, ProcessingJob } from "./types";
import { apiClient } from "./api/client";
import { LandingPage } from "./components/LandingPage";
import { UploadInterface } from "./components/UploadInterface";
import { ProcessingView } from "./components/ProcessingView";
import { ResultDisplay } from "./components/ResultDisplay";

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
      setUserPhotoProgress(0);
      setState((prev) => ({ ...prev, error: null }));

      // Simulate progress (in real implementation, use XMLHttpRequest for progress tracking)
      const progressInterval = setInterval(() => {
        setUserPhotoProgress((prev) => Math.min(prev + 10, 90));
      }, 100);

      const response = await apiClient.uploadUserPhoto(file);

      clearInterval(progressInterval);
      setUserPhotoProgress(100);

      const uploadedImage: UploadedImage = {
        file,
        fileId: response.fileId,
        previewUrl: apiClient.getPreviewUrl(response.fileId),
      };
      setState((prev) => ({ ...prev, userPhoto: uploadedImage }));

      // Reset progress after a short delay
      setTimeout(() => setUserPhotoProgress(0), 1000);
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
      setShirtProgress(0);
      setState((prev) => ({ ...prev, error: null }));

      // Simulate progress (in real implementation, use XMLHttpRequest for progress tracking)
      const progressInterval = setInterval(() => {
        setShirtProgress((prev) => Math.min(prev + 10, 90));
      }, 100);

      const response = await apiClient.uploadShirt(file);

      clearInterval(progressInterval);
      setShirtProgress(100);

      const uploadedImage: UploadedImage = {
        file,
        fileId: response.fileId,
        previewUrl: apiClient.getPreviewUrl(response.fileId),
      };
      setState((prev) => ({ ...prev, shirtImage: uploadedImage }));

      // Reset progress after a short delay
      setTimeout(() => setShirtProgress(0), 1000);
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

  return (
    <Router>
      <div className="App">
        <header className="App-header">
          <h1>Virtual FitCheck</h1>
          <p>AI-Powered Virtual Try-On</p>
        </header>
        <main className="App-main">
          <Routes>
            <Route
              path="/"
              element={
                <>
                  {/* Show landing page first */}
                  {showLanding && (
                    <LandingPage onGetStarted={handleGetStarted} />
                  )}

                  {/* Show upload interface when not processing and no result */}
                  {!showLanding && !state.processingJob && (
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
                          <button
                            className="retry-button"
                            onClick={handleRetry}
                          >
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
                  )}

                  {/* Show processing view when job is in progress */}
                  {!showLanding &&
                    state.processingJob &&
                    state.processingJob.status !== "completed" && (
                      <ProcessingView
                        jobId={state.processingJob.jobId}
                        onComplete={handleProcessingComplete}
                        onError={handleProcessingError}
                      />
                    )}

                  {/* Show result display when processing is complete */}
                  {!showLanding &&
                    state.processingJob &&
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
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;
