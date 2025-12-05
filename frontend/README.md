# Virtual FitCheck Frontend

React + TypeScript frontend for the Virtual FitCheck application.

## Setup

1. Install dependencies:

```bash
npm install --legacy-peer-deps
```

2. Configure environment variables:

```bash
cp .env.example .env
```

Edit `.env` to set the backend API URL (default: `http://localhost:8080`)

3. Start the development server:

```bash
npm start
```

The app will open at `http://localhost:3000`

## Project Structure

```
src/
├── api/
│   └── client.ts          # API client for backend communication
├── types/
│   └── index.ts           # TypeScript type definitions
├── App.tsx                # Main application component with routing
├── App.css                # Application styles
├── index.tsx              # Application entry point
└── index.css              # Global styles
```

## API Client

The API client (`src/api/client.ts`) provides methods for:

- `uploadUserPhoto(file)` - Upload user photo
- `uploadShirt(file)` - Upload shirt image
- `processImages(userPhotoId, shirtImageId)` - Initiate processing
- `getJobStatus(jobId)` - Check processing status
- `getPreviewUrl(fileId)` - Get preview URL for uploaded file
- `getResultUrl(resultId)` - Get result download URL

## Available Scripts

- `npm start` - Run development server
- `npm test` - Run tests
- `npm run build` - Build for production
- `npm run eject` - Eject from Create React App (one-way operation)

## Environment Variables

- `REACT_APP_API_URL` - Backend API base URL (default: `http://localhost:8080`)
