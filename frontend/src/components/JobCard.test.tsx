import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import "@testing-library/jest-dom";
import { JobCard, Job } from "./JobCard";

describe("JobCard Component", () => {
  const mockOnDelete = jest.fn();
  const mockOnView = jest.fn();

  beforeEach(() => {
    mockOnDelete.mockClear();
    mockOnView.mockClear();
  });

  const createMockJob = (overrides?: Partial<Job>): Job => ({
    jobId: "test-job-id",
    status: "completed",
    progress: 100,
    statusMessage: "Processing completed",
    resultUrl: "http://localhost:8080/api/result/test-result",
    thumbnailUrl: "http://localhost:8080/api/preview/test-thumbnail",
    createdAt: new Date().toISOString(),
    ...overrides,
  });

  it("should render job card with thumbnail", () => {
    const job = createMockJob();
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const thumbnail = container.querySelector(".job-thumbnail img");
    expect(thumbnail).toBeInTheDocument();
    expect(thumbnail).toHaveAttribute("src", job.thumbnailUrl);
  });

  it("should render placeholder when no thumbnail is provided", () => {
    const job = createMockJob({ thumbnailUrl: undefined });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const placeholder = container.querySelector(".thumbnail-placeholder");
    expect(placeholder).toBeInTheDocument();
  });

  it("should display job status with correct label", () => {
    const statuses: Array<Job["status"]> = [
      "pending",
      "processing",
      "completed",
      "failed",
    ];
    const expectedLabels = ["Queued", "Processing", "Completed", "Failed"];

    statuses.forEach((status, index) => {
      const job = createMockJob({ status });
      const { container } = render(
        <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
      );

      const statusLabel = container.querySelector(".status-label");
      expect(statusLabel).toHaveTextContent(expectedLabels[index]);
    });
  });

  it("should display progress bar for pending and processing jobs", () => {
    const job = createMockJob({ status: "processing", progress: 50 });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const progressBar = container.querySelector(".job-progress");
    expect(progressBar).toBeInTheDocument();

    const progressText = container.querySelector(".progress-text");
    expect(progressText).toHaveTextContent("50%");
  });

  it("should not display progress bar for completed jobs", () => {
    const job = createMockJob({ status: "completed" });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const progressBar = container.querySelector(".job-progress");
    expect(progressBar).not.toBeInTheDocument();
  });

  it("should display View button for completed jobs with result URL", () => {
    const job = createMockJob({
      status: "completed",
      resultUrl: "http://localhost:8080/api/result/test",
    });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const viewButton = container.querySelector(".view-button");
    expect(viewButton).toBeInTheDocument();
    expect(viewButton).toHaveTextContent("View");
  });

  it("should not display View button for non-completed jobs", () => {
    const job = createMockJob({ status: "processing", resultUrl: undefined });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const viewButton = container.querySelector(".view-button");
    expect(viewButton).not.toBeInTheDocument();
  });

  it("should call onView when View button is clicked", () => {
    const job = createMockJob({ status: "completed" });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const viewButton = container.querySelector(".view-button");
    fireEvent.click(viewButton!);

    expect(mockOnView).toHaveBeenCalledWith(job.jobId);
    expect(mockOnView).toHaveBeenCalledTimes(1);
  });

  it("should display Delete button for all jobs", () => {
    const job = createMockJob();
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const deleteButton = container.querySelector(".delete-button");
    expect(deleteButton).toBeInTheDocument();
    expect(deleteButton).toHaveTextContent("Delete");
  });

  it("should show confirmation dialog when Delete button is clicked", () => {
    const job = createMockJob();
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const deleteButton = container.querySelector(".delete-button");
    fireEvent.click(deleteButton!);

    const modal = container.querySelector(".modal-overlay");
    expect(modal).toBeInTheDocument();

    const modalContent = container.querySelector(".modal-content");
    expect(modalContent).toHaveTextContent("Delete Job?");
  });

  it("should call onDelete when deletion is confirmed", () => {
    const job = createMockJob();
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    // Click delete button
    const deleteButton = container.querySelector(".delete-button");
    fireEvent.click(deleteButton!);

    // Confirm deletion
    const confirmButton = container.querySelector(".btn-delete");
    fireEvent.click(confirmButton!);

    expect(mockOnDelete).toHaveBeenCalledWith(job.jobId);
    expect(mockOnDelete).toHaveBeenCalledTimes(1);
  });

  it("should not call onDelete when deletion is cancelled", () => {
    const job = createMockJob();
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    // Click delete button
    const deleteButton = container.querySelector(".delete-button");
    fireEvent.click(deleteButton!);

    // Cancel deletion
    const cancelButton = container.querySelector(".btn-cancel");
    fireEvent.click(cancelButton!);

    expect(mockOnDelete).not.toHaveBeenCalled();
  });

  it("should close modal when clicking outside", () => {
    const job = createMockJob();
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    // Click delete button to open modal
    const deleteButton = container.querySelector(".delete-button");
    fireEvent.click(deleteButton!);

    // Click overlay
    const overlay = container.querySelector(".modal-overlay");
    fireEvent.click(overlay!);

    // Modal should be closed (not in document)
    const modalAfterClick = container.querySelector(".modal-overlay");
    expect(modalAfterClick).not.toBeInTheDocument();
  });

  it("should display error message for failed jobs", () => {
    const errorMessage = "Processing failed due to invalid image";
    const job = createMockJob({
      status: "failed",
      error: errorMessage,
    });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const errorElement = container.querySelector(".job-error");
    expect(errorElement).toBeInTheDocument();
    expect(errorElement).toHaveTextContent(errorMessage);
  });

  it("should display status message for non-failed jobs", () => {
    const statusMessage = "Analyzing image...";
    const job = createMockJob({
      status: "processing",
      statusMessage,
    });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const messageElement = container.querySelector(".job-message");
    expect(messageElement).toBeInTheDocument();
    expect(messageElement).toHaveTextContent(statusMessage);
  });

  it("should format creation time correctly", () => {
    const now = new Date();
    const job = createMockJob({ createdAt: now.toISOString() });
    const { container } = render(
      <JobCard job={job} onDelete={mockOnDelete} onView={mockOnView} />
    );

    const timeElement = container.querySelector(".job-time");
    expect(timeElement).toBeInTheDocument();
    expect(timeElement?.textContent).toBeTruthy();
  });
});
