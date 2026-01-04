import React from "react";
import { useNavigate } from "react-router-dom";
import "./LandingPage.css";

interface LandingPageProps {
  onGetStarted?: () => void;
}

export const LandingPage: React.FC<LandingPageProps> = ({ onGetStarted }) => {
  const navigate = useNavigate();
  return (
    <div className="landing-page">
      <div className="landing-container">
        {/* Hero Section */}
        <div className="hero-section">
          <div className="hero-badge">
            <span className="badge-icon">✨</span>
            <span>AI-Powered Virtual Try-On</span>
          </div>

          <h1 className="hero-title">
            Try On Clothes
            <br />
            <span className="gradient-text">Without Leaving Home</span>
          </h1>

          <p className="hero-description">
            Upload your photo and a shirt image to see how it looks on you.
            Powered by advanced AI technology for realistic virtual try-ons.
          </p>

          <button
            className="cta-button"
            onClick={() => {
              if (onGetStarted) onGetStarted();
              navigate("/upload");
            }}
          >
            <span>Get Started</span>
            <svg
              className="arrow-icon"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
          </button>
        </div>

        {/* Features Section */}
        <div className="features-section">
          <div className="feature-card">
            <div className="feature-icon">🚀</div>
            <h3 className="feature-title">Fast & Easy</h3>
            <p className="feature-description">
              Upload your photos and get results in seconds
            </p>
          </div>

          <div className="feature-card">
            <div className="feature-icon">🎯</div>
            <h3 className="feature-title">Accurate Results</h3>
            <p className="feature-description">
              AI-powered technology for realistic try-ons
            </p>
          </div>

          <div className="feature-card">
            <div className="feature-icon">🔒</div>
            <h3 className="feature-title">Private & Secure</h3>
            <p className="feature-description">
              Your photos are processed securely and deleted after use
            </p>
          </div>
        </div>

        {/* How It Works */}
        <div className="how-it-works">
          <h2 className="section-title">How It Works</h2>

          <div className="steps">
            <div className="step">
              <div className="step-number">1</div>
              <h4 className="step-title">Upload Your Photo</h4>
              <p className="step-description">
                Take or upload a clear photo of yourself
              </p>
            </div>

            <div className="step-arrow">→</div>

            <div className="step">
              <div className="step-number">2</div>
              <h4 className="step-title">Choose a Shirt</h4>
              <p className="step-description">
                Upload an image of the shirt you want to try
              </p>
            </div>

            <div className="step-arrow">→</div>

            <div className="step">
              <div className="step-number">3</div>
              <h4 className="step-title">See the Result</h4>
              <p className="step-description">
                Get your virtual try-on result instantly
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
