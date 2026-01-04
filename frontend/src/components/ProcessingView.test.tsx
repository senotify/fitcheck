import React from "react";
import { render, screen, waitFor, act } from "@testing-library/react";
import "@testing-library/jest-dom";
import * as fc from "fast-check";
import { ProcessingView } from "./ProcessingView";
import { apiClient } from "../api/client";

// Mock the API client
jest.mock("../api/client", () => ({
  apiClient: {
    getJobStatus: jest.fn(),
    notifyEmail: jest.fn(),
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

// Feature: virtual-fitcheck, Property 26: Long processing notification
// For any processing job that exceeds 3 minutes, the system should display a notification to the user informing them that processing is taking longer than expected.
// Validates: Requirements 9.1
describe("Property 26: Long processing notification", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it("should display long processing notification after 3 minutes for any processing job", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.record({
          jobId: fc.uuid(),
          progress: fc.integer({ min: 0, max: 99 }),
          message: fc.constantFrom(
            "Processing your image...",
            "Applying shirt to photo...",
            "Generating result...",
            "Still working on it..."
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

          // Wait for initial API call
          await waitFor(() => {
            expect(mockedApiClient.getJobStatus).toHaveBeenCalledWith(
              jobData.jobId
            );
          });

          // Property: Modal should NOT be visible before 3 minutes
          let modal = container.querySelector(".modal-overlay");
          expect(modal).not.toBeInTheDocument();

          // Fast-forward time to 3 minutes (180 seconds)
          await act(async () => {
            jest.advanceTimersByTime(180000);
          });

          // Property: Modal SHOULD be visible after 3 minutes
          modal = container.querySelector(".modal-overlay");
          expect(modal).toBeInTheDocument();

          // Verify modal content
          const modalContent = container.querySelector(".modal-content");
          expect(modalContent).toBeInTheDocument();
          expect(modalContent).toHaveTextContent(
            "Processing is taking longer than expected"
          );

          // Clean up
          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  it("should not display notification if processing completes before 3 minutes", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.record({
          jobId: fc.uuid(),
          resultUrl: fc.webUrl(),
        }),
        async (jobData) => {
          const mockOnComplete = jest.fn();
          const mockOnError = jest.fn();

          // Mock API response that completes quickly
          mockedApiClient.getJobStatus.mockResolvedValue({
            status: "completed",
            progress: 100,
            message: "Processing complete!",
            resultUrl: jobData.resultUrl,
          });

          const { container, unmount } = render(
            <ProcessingView
              jobId={jobData.jobId}
              onComplete={mockOnComplete}
              onError={mockOnError}
            />
          );

          // Wait for completion
          await waitFor(() => {
            expect(mockOnComplete).toHaveBeenCalledWith(jobData.resultUrl);
          });

          // Fast-forward time to 3 minutes
          await act(async () => {
            jest.advanceTimersByTime(180000);
          });

          // Property: Modal should NOT appear if job completed
          const modal = container.querySelector(".modal-overlay");
          expect(modal).not.toBeInTheDocument();

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });
});

// Feature: virtual-fitcheck, Property 27: Email notification option availability
// For any processing job that exceeds 3 minutes, the system should present the user with options to either receive results via email or continue waiting.
// Validates: Requirements 9.2
describe("Property 27: Email notification option availability", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it("should present both email and continue waiting options after 3 minutes", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.record({
          jobId: fc.uuid(),
          progress: fc.integer({ min: 0, max: 99 }),
          message: fc.string(),
        }),
        async (jobData) => {
          const mockOnComplete = jest.fn();
          const mockOnError = jest.fn();

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

          await waitFor(() => {
            expect(mockedApiClient.getJobStatus).toHaveBeenCalled();
          });

          // Fast-forward to 3 minutes
          await act(async () => {
            jest.advanceTimersByTime(180000);
          });

          // Property: Both options should be available
          const modal = container.querySelector(".modal-overlay");
          expect(modal).toBeInTheDocument();

          // Email option should be present
          const emailOption = container.querySelector(".email-option");
          expect(emailOption).toBeInTheDocument();
          expect(emailOption).toHaveTextContent("Get notified by email");

          const emailInput = container.querySelector(
            'input[type="email"]'
          ) as HTMLInputElement;
          expect(emailInput).toBeInTheDocument();

          const emailButton = container.querySelector(
            ".email-option .btn-primary"
          );
          expect(emailButton).toBeInTheDocument();
          expect(emailButton).toHaveTextContent("Send me an email");

          // Continue waiting option should be present
          const continueOption = container.querySelector(".continue-option");
          expect(continueOption).toBeInTheDocument();
          expect(continueOption).toHaveTextContent("Keep waiting");

          const continueButton = container.querySelector(
            ".continue-option .btn-secondary"
          );
          expect(continueButton).toBeInTheDocument();
          expect(continueButton).toHaveTextContent("Continue waiting");

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  it("should have functional email input field", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.record({
          jobId: fc.uuid(),
        }),
        async (jobData) => {
          const mockOnComplete = jest.fn();
          const mockOnError = jest.fn();

          mockedApiClient.getJobStatus.mockResolvedValue({
            status: "processing",
            progress: 50,
            message: "Processing...",
          });

          const { container, unmount } = render(
            <ProcessingView
              jobId={jobData.jobId}
              onComplete={mockOnComplete}
              onError={mockOnError}
            />
          );

          await waitFor(() => {
            expect(mockedApiClient.getJobStatus).toHaveBeenCalled();
          });

          // Fast-forward to 3 minutes
          await act(async () => {
            jest.advanceTimersByTime(180000);
          });

          // Property: Email input should be functional
          const emailInput = container.querySelector(
            'input[type="email"]'
          ) as HTMLInputElement;
          expect(emailInput).toBeInTheDocument();
          expect(emailInput.placeholder).toBeTruthy();

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });
});

// Feature: virtual-fitcheck, Property 30: Continued progress updates
// For any user who chooses to continue waiting beyond 3 minutes, the system should continue displaying progress updates at regular intervals.
// Validates: Requirements 9.5
describe("Property 30: Continued progress updates", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it("should continue polling and updating progress after user dismisses modal", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.record({
          jobId: fc.uuid(),
          initialProgress: fc.integer({ min: 0, max: 50 }),
          updatedProgress: fc.integer({ min: 51, max: 99 }),
          initialMessage: fc.constantFrom(
            "Processing...",
            "Working on it...",
            "In progress..."
          ),
          updatedMessage: fc.constantFrom(
            "Almost done...",
            "Finalizing...",
            "Nearly complete..."
          ),
        }),
        async (jobData) => {
          const mockOnComplete = jest.fn();
          const mockOnError = jest.fn();

          // Initial API response
          mockedApiClient.getJobStatus.mockResolvedValueOnce({
            status: "processing",
            progress: jobData.initialProgress,
            message: jobData.initialMessage,
          });

          const { container, unmount } = render(
            <ProcessingView
              jobId={jobData.jobId}
              onComplete={mockOnComplete}
              onError={mockOnError}
            />
          );

          await waitFor(() => {
            expect(mockedApiClient.getJobStatus).toHaveBeenCalled();
          });

          // Fast-forward to 3 minutes to show modal
          await act(async () => {
            jest.advanceTimersByTime(180000);
          });

          // Verify modal is shown
          let modal = container.querySelector(".modal-overlay");
          expect(modal).toBeInTheDocument();

          // Mock updated API response for next poll
          mockedApiClient.getJobStatus.mockResolvedValue({
            status: "processing",
            progress: jobData.updatedProgress,
            message: jobData.updatedMessage,
          });

          // Click "Continue waiting" button
          const continueButton = container.querySelector(
            ".continue-option .btn-secondary"
          ) as HTMLButtonElement;
          expect(continueButton).toBeInTheDocument();

          await act(async () => {
            continueButton.click();
          });

          // Property: Modal should be dismissed
          modal = container.querySelector(".modal-overlay");
          expect(modal).not.toBeInTheDocument();

          // Property: Polling should continue - advance time for next poll (5 seconds)
          const callCountBefore =
            mockedApiClient.getJobStatus.mock.calls.length;

          await act(async () => {
            jest.advanceTimersByTime(5000);
          });

          // Wait for the API call to complete
          await waitFor(() => {
            expect(
              mockedApiClient.getJobStatus.mock.calls.length
            ).toBeGreaterThan(callCountBefore);
          });

          // Property: Progress should be updated
          await waitFor(() => {
            const progressText = container.querySelector(".progress-text");
            expect(progressText).toHaveTextContent(
              `${jobData.updatedProgress}%`
            );
          });

          // Property: Status message should be updated
          await waitFor(() => {
            const statusMessage = container.querySelector(".status-message");
            expect(statusMessage).toHaveTextContent(jobData.updatedMessage);
          });

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });

  it("should continue updating elapsed time after dismissing modal", async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.record({
          jobId: fc.uuid(),
        }),
        async (jobData) => {
          const mockOnComplete = jest.fn();
          const mockOnError = jest.fn();

          mockedApiClient.getJobStatus.mockResolvedValue({
            status: "processing",
            progress: 50,
            message: "Processing...",
          });

          const { container, unmount } = render(
            <ProcessingView
              jobId={jobData.jobId}
              onComplete={mockOnComplete}
              onError={mockOnError}
            />
          );

          await waitFor(() => {
            expect(mockedApiClient.getJobStatus).toHaveBeenCalled();
          });

          // Fast-forward to 3 minutes
          await act(async () => {
            jest.advanceTimersByTime(180000);
          });

          // Dismiss modal
          const continueButton = container.querySelector(
            ".continue-option .btn-secondary"
          ) as HTMLButtonElement;

          await act(async () => {
            continueButton.click();
          });

          // Property: Elapsed time should continue updating
          let elapsedTime = container.querySelector(".elapsed-time");
          expect(elapsedTime).toHaveTextContent("3:00"); // 3 minutes

          // Advance another minute
          await act(async () => {
            jest.advanceTimersByTime(60000);
          });

          elapsedTime = container.querySelector(".elapsed-time");
          expect(elapsedTime).toHaveTextContent("4:00"); // 4 minutes

          unmount();
        }
      ),
      { numRuns: 100 }
    );
  });
});
