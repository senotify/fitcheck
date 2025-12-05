import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import "@testing-library/jest-dom";
import * as fc from "fast-check";
import { ProcessingView } from "./ProcessingView";
import { apiClient } from "../api/client";

// Mock the API client
jest.mock("../api/client", () => ({
  apiClient: {
    getJobStatus: jest.fn(),
  },
}));

const mockedApiClient = apiClient as jest.Mocked<typeof apiClient>;

// Feature: virtual-fitcheck, Property 7: Loading state display
// For any processing job with status "processing", the UI should display a loading indicator.
// Validates: Requirements 3.3
describe("Property 7: Loading state display", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it("should display loading indicator for any job with processing status", async () => {
    await fc.assert(
      fc.asyncProperty(
        // Generate arbitrary job data with "processing" status
        fc.record({
          jobId: fc.uuid(),
          progress: fc.integer({ min: 0, max: 100 }),
          message: fc.constantFrom(
            "Processing your image...",
            "Applying shirt to photo...",
            "Generating result...",
            "Almost done..."
          ),
        }),
        async (jobData) => {
          const mockOnComplete = jest.fn();
          const mockOnError = jest.fn();

          // Mock API response with "processing" status
          mockedApiClient.getJobStatus.mockResolvedValue({
            status: "processing",
            progress: jobData.progress,
            message: jobData.message,
          });

          const { container, unmount } = render(
            <ProcessingView
              jobId={jobData.jobId}
              onComplete={mockOnComplete}
              onError={mockOnError}
            />
          );

          // Wait for initial API call and state update
          await waitFor(() => {
            expect(mockedApiClient.getJobStatus).toHaveBeenCalledWith(
              jobData.jobId
            );
          });

          // Wait for the component to update with the API response
          await waitFor(() => {
            const statusMessage = container.querySelector(".status-message");
            expect(statusMessage).toHaveTextContent(jobData.message);
          });

          // Property: Loading indicator should be present when status is "processing"
          const loadingAnimation =
            container.querySelector(".loading-animation");
          expect(loadingAnimation).toBeInTheDocument();

          // Verify spinner is present
          const spinner = container.querySelector(".spinner");
          expect(spinner).toBeInTheDocument();

          // Verify pulse ring is present
          const pulseRing = container.querySelector(".pulse-ring");
          expect(pulseRing).toBeInTheDocument();

          // Verify progress is displayed within the container
          const progressText = container.querySelector(".progress-text");
          expect(progressText).toHaveTextContent(`${jobData.progress}%`);

          // Clean up after each test iteration
          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  it("should display loading indicator for any job with pending status", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.record({
          jobId: fc.uuid(),
          progress: fc.integer({ min: 0, max: 100 }),
          message: fc.constantFrom(
            "Queued for processing...",
            "Waiting to start...",
            "Preparing..."
          ),
        }),
        async (jobData) => {
          const mockOnComplete = jest.fn();
          const mockOnError = jest.fn();

          // Mock API response with "pending" status
          mockedApiClient.getJobStatus.mockResolvedValue({
            status: "pending",
            progress: jobData.progress,
            message: jobData.message,
          });

          const { container } = render(
            <ProcessingView
              jobId={jobData.jobId}
              onComplete={mockOnComplete}
              onError={mockOnError}
            />
          );

          // Wait for initial API call
          await waitFor(() => {
            expect(mockedApiClient.getJobStatus).toHaveBeenCalledWith(
              jobData.jobId
            );
          });

          // Property: Loading indicator should be present when status is "pending"
          const loadingAnimation =
            container.querySelector(".loading-animation");
          expect(loadingAnimation).toBeInTheDocument();

          const spinner = container.querySelector(".spinner");
          expect(spinner).toBeInTheDocument();

          const pulseRing = container.querySelector(".pulse-ring");
          expect(pulseRing).toBeInTheDocument();
        }
      ),
      { numRuns: 100 }
    );
  });

  it("should not call onComplete or onError while processing", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.record({
          jobId: fc.uuid(),
          progress: fc.integer({ min: 0, max: 99 }), // Not 100% yet
          message: fc.constantFrom(
            "Processing...",
            "Working on it...",
            "In progress..."
          ),
        }),
        async (jobData) => {
          const mockOnComplete = jest.fn();
          const mockOnError = jest.fn();

          // Mock API response with "processing" status
          mockedApiClient.getJobStatus.mockResolvedValue({
            status: "processing",
            progress: jobData.progress,
            message: jobData.message,
          });

          render(
            <ProcessingView
              jobId={jobData.jobId}
              onComplete={mockOnComplete}
              onError={mockOnError}
            />
          );

          // Wait for initial API call
          await waitFor(() => {
            expect(mockedApiClient.getJobStatus).toHaveBeenCalledWith(
              jobData.jobId
            );
          });

          // Property: While processing, neither onComplete nor onError should be called
          expect(mockOnComplete).not.toHaveBeenCalled();
          expect(mockOnError).not.toHaveBeenCalled();
        }
      ),
      { numRuns: 100 }
    );
  });
});
