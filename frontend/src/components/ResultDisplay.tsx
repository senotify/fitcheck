import React from "react";
import "./ResultDisplay.css";

interface ResultDisplayProps {
  resultUrl: string;
  onDownload: () => void;
  onTryAnother: () => void;
}

export const ResultDisplay: React.FC<ResultDisplayProps> = ({
  resultUrl,
  onDownload,
  onTryAnother,
}) => {
  return (
    <div className="result-display">
      <div className="result-container">
        <h2 className="result-title">Your Virtual Try-On Result</h2>
        <p className="result-subtitle">Here's how the shirt looks on you!</p>

        {/* Composite Image Display */}
        <div className="result-image-container">
          <img
            src={resultUrl}
            alt="Virtual try-on result"
            className="result-image"
          />
        </div>

        {/* Action Buttons */}
        <div className="result-actions">
          <button
            className="download-button"
            onClick={onDownload}
            aria-label="Download result image"
          >
            <svg
              className="button-icon"
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
            className="try-another-button"
            onClick={onTryAnother}
            aria-label="Try another shirt"
          >
            <svg
              className="button-icon"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" />
              <path d="M21 3v5h-5" />
              <path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" />
              <path d="M3 21v-5h5" />
            </svg>
            Try Another Shirt
          </button>
        </div>
      </div>
    </div>
  );
};
