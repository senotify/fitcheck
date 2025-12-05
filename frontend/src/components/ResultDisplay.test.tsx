import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import "@testing-library/jest-dom";
import * as fc from "fast-check";
import { ResultDisplay } from "./ResultDisplay";

// Feature: virtual-fitcheck, Property 8: Successful result display
// For any processing job that completes with status "completed", the system should display the composite image to the user.
// Validates: Requirements 3.4
describe("Property 8: Successful result display", () => {
  it("should display composite image for any completed job result", () => {
    fc.assert(
      fc.property(
        // Generate arbitrary result URLs
        fc.record({
          resultId: fc.uuid(),
          baseUrl: fc.constantFrom(
            "http://localhost:8080",
            "https://api.example.com",
            "https://fitcheck.app"
          ),
        }),
        (data) => {
          const resultUrl = `${data.baseUrl}/api/result/${data.resultId}`;
          const mockOnDownload = jest.fn();
          const mockOnTryAnother = jest.fn();

          const { container } = render(
            <ResultDisplay
              resultUrl={resultUrl}
              onDownload={mockOnDownload}
              onTryAnother={mockOnTryAnother}
            />
          );

          // Property: Composite image should be displayed
          const image = container.querySelector(".result-image");
          expect(image).toBeInTheDocument();
          expect(image).toHaveAttribute("src", resultUrl);
          expect(image).toHaveAttribute("alt", "Virtual try-on result");

          // Verify the image is within the result container
          const imageContainer = container.querySelector(
            ".result-image-container"
          );
          expect(imageContainer).toBeInTheDocument();
          expect(imageContainer).toContainElement(image as HTMLElement);
        }
      ),
      { numRuns: 100 }
    );
  });

  it("should display result with proper title and subtitle", () => {
    fc.assert(
      fc.property(
        fc.webUrl(), // Generate arbitrary URLs
        (resultUrl) => {
          const mockOnDownload = jest.fn();
          const mockOnTryAnother = jest.fn();

          const { container } = render(
            <ResultDisplay
              resultUrl={resultUrl}
              onDownload={mockOnDownload}
              onTryAnother={mockOnTryAnother}
            />
          );

          // Property: Result display should have descriptive title and subtitle
          const title = container.querySelector(".result-title");
          expect(title).toBeInTheDocument();
          expect(title).toHaveTextContent(/virtual try-on result/i);

          const subtitle = container.querySelector(".result-subtitle");
          expect(subtitle).toBeInTheDocument();
          expect(subtitle?.textContent).toBeTruthy();
        }
      ),
      { numRuns: 100 }
    );
  });
});

// Feature: virtual-fitcheck, Property 10: Download availability
// For any successfully generated composite image, the system should provide a download button in the UI.
// Validates: Requirements 4.1
describe("Property 10: Download availability", () => {
  it("should provide download button for any result URL", () => {
    fc.assert(
      fc.property(
        fc.webUrl(), // Generate arbitrary result URLs
        (resultUrl) => {
          const mockOnDownload = jest.fn();
          const mockOnTryAnother = jest.fn();

          const { container } = render(
            <ResultDisplay
              resultUrl={resultUrl}
              onDownload={mockOnDownload}
              onTryAnother={mockOnTryAnother}
            />
          );

          // Property: Download button should be present
          const downloadButton = container.querySelector(".download-button");
          expect(downloadButton).toBeInTheDocument();
          expect(downloadButton).toHaveTextContent(/download/i);

          // Verify button has proper accessibility
          expect(downloadButton).toHaveAttribute(
            "aria-label",
            "Download result image"
          );
        }
      ),
      { numRuns: 100 }
    );
  });

  it("should call onDownload when download button is clicked", () => {
    fc.assert(
      fc.property(
        fc.webUrl(),
        fc.integer({ min: 1, max: 10 }), // Number of clicks
        (resultUrl, numClicks) => {
          const mockOnDownload = jest.fn();
          const mockOnTryAnother = jest.fn();

          const { container } = render(
            <ResultDisplay
              resultUrl={resultUrl}
              onDownload={mockOnDownload}
              onTryAnother={mockOnTryAnother}
            />
          );

          const downloadButton = container.querySelector(".download-button");
          expect(downloadButton).toBeInTheDocument();

          // Click the button multiple times
          for (let i = 0; i < numClicks; i++) {
            fireEvent.click(downloadButton!);
          }

          // Property: onDownload should be called for each click
          expect(mockOnDownload).toHaveBeenCalledTimes(numClicks);
        }
      ),
      { numRuns: 100 }
    );
  });

  it("should provide try another button for any result", () => {
    fc.assert(
      fc.property(fc.webUrl(), (resultUrl) => {
        const mockOnDownload = jest.fn();
        const mockOnTryAnother = jest.fn();

        const { container } = render(
          <ResultDisplay
            resultUrl={resultUrl}
            onDownload={mockOnDownload}
            onTryAnother={mockOnTryAnother}
          />
        );

        // Property: Try another button should be present
        const tryAnotherButton = container.querySelector(".try-another-button");
        expect(tryAnotherButton).toBeInTheDocument();
        expect(tryAnotherButton).toHaveTextContent(/try another/i);

        // Verify button has proper accessibility
        expect(tryAnotherButton).toHaveAttribute(
          "aria-label",
          "Try another shirt"
        );
      }),
      { numRuns: 100 }
    );
  });

  it("should call onTryAnother when try another button is clicked", () => {
    fc.assert(
      fc.property(
        fc.webUrl(),
        fc.integer({ min: 1, max: 10 }),
        (resultUrl, numClicks) => {
          const mockOnDownload = jest.fn();
          const mockOnTryAnother = jest.fn();

          const { container } = render(
            <ResultDisplay
              resultUrl={resultUrl}
              onDownload={mockOnDownload}
              onTryAnother={mockOnTryAnother}
            />
          );

          const tryAnotherButton = container.querySelector(
            ".try-another-button"
          );
          expect(tryAnotherButton).toBeInTheDocument();

          // Click the button multiple times
          for (let i = 0; i < numClicks; i++) {
            fireEvent.click(tryAnotherButton!);
          }

          // Property: onTryAnother should be called for each click
          expect(mockOnTryAnother).toHaveBeenCalledTimes(numClicks);
        }
      ),
      { numRuns: 100 }
    );
  });
});
