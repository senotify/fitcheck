# Virtual FitCheck

A web application that enables users to virtually try on clothing using generative AI. Upload a photo of yourself and an image of a shirt to see how it would look on you.

## Features

- **Image Upload**: Upload photos of yourself and shirt images
- **AI-Powered Virtual Try-On**: Uses generative AI to composite clothing onto your photo
- **Real-time Processing**: Track processing status with live updates
- **Download Results**: Save your virtual try-on results
- **Multiple Try-Ons**: Try different shirts with the same photo without re-uploading
- **Responsive Design**: Works on desktop and mobile devices

## Architecture

- **Frontend**: React with TypeScript
- **Backend**: Go with Gin framework
- **AI Integration**: Replicate API (or compatible service)
- **Testing**: Property-based testing with gopter (Go) and fast-check (TypeScript)

## Prerequisites

### Backend

- Go 1.21 or higher
- AI service API key (Replicate or compatible)

### Frontend

- Node.js 16 or higher
- npm or yarn

## Setup Instructions

### Backend Setup

1. Navigate to the backend directory:

```bash
cd backend
```

2. Install dependencies:

```bash
go mod download
```

3. Create a `.env` file (copy from `.env.example`):

```bash
cp .env.example .env
```

4. Configure environment variables in `.env`:

```env
PORT=8080
STORAGE_PATH=./tmp/uploads
AI_SERVICE_URL=https://api.replicate.com/v1/predictions
AI_SERVICE_KEY=your_api_key_here
```

5. Run the backend server:

```bash
go run main.go
```

The backend will start on `http://localhost:8080`

### Frontend Setup

1. Navigate to the frontend directory:

```bash
cd frontend
```

2. Install dependencies:

```bash
npm install
```

3. Create a `.env` file (copy from `.env.example`):

```bash
cp .env.example .env
```

4. Configure the API URL in `.env`:

```env
REACT_APP_API_URL=http://localhost:8080
```

5. Start the development server:

```bash
npm start
```

The frontend will start on `http://localhost:3000`

## Running Tests

### Backend Tests

```bash
cd backend
go test ./... -v
```

### Frontend Tests

```bash
cd frontend
npm test
```

## API Documentation

### Upload Endpoints

#### POST /api/upload/user-photo

Upload a user photo for virtual try-on.

**Request:**

- Content-Type: `multipart/form-data`
- Body: `image` file (JPEG, PNG, or WebP, max 10MB)

**Response:**

```json
{
  "success": true,
  "fileId": "uuid-string",
  "previewUrl": "/api/preview/uuid-string"
}
```

#### POST /api/upload/shirt

Upload a shirt image for virtual try-on.

**Request:**

- Content-Type: `multipart/form-data`
- Body: `image` file (JPEG, PNG, or WebP, max 10MB)

**Response:**

```json
{
  "success": true,
  "fileId": "uuid-string",
  "previewUrl": "/api/preview/uuid-string"
}
```

### Processing Endpoints

#### POST /api/process

Initiate virtual try-on processing.

**Request:**

```json
{
  "userPhotoId": "uuid-string",
  "shirtImageId": "uuid-string"
}
```

**Response:**

```json
{
  "success": true,
  "jobId": "uuid-string",
  "estimatedTime": 15
}
```

#### GET /api/status/:jobId

Check processing status.

**Response:**

```json
{
  "status": "processing",
  "progress": 75,
  "message": "Applying shirt to image...",
  "resultUrl": "/api/result/uuid-string"
}
```

Status values: `pending`, `processing`, `completed`, `failed`

### Result Endpoints

#### GET /api/preview/:fileId

Get preview of uploaded image.

**Response:** Image binary data

#### GET /api/result/:resultId

Download the composite result image.

**Response:** Image binary data with download headers

## Environment Variables

### Backend

| Variable         | Description                 | Default         |
| ---------------- | --------------------------- | --------------- |
| `PORT`           | Server port                 | `8080`          |
| `STORAGE_PATH`   | Temporary file storage path | `./tmp/uploads` |
| `AI_SERVICE_URL` | AI service API endpoint     | -               |
| `AI_SERVICE_KEY` | AI service API key          | -               |

### Frontend

| Variable            | Description     | Default                 |
| ------------------- | --------------- | ----------------------- |
| `REACT_APP_API_URL` | Backend API URL | `http://localhost:8080` |

## Project Structure

```
virtual-fitcheck/
├── backend/
│   ├── internal/
│   │   ├── api/          # HTTP handlers and routing
│   │   ├── config/       # Configuration management
│   │   ├── logger/       # Logging utilities
│   │   ├── models/       # Data models
│   │   └── services/     # Business logic services
│   ├── main.go           # Application entry point
│   └── go.mod            # Go dependencies
├── frontend/
│   ├── src/
│   │   ├── api/          # API client
│   │   ├── components/   # React components
│   │   ├── types/        # TypeScript types
│   │   ├── App.tsx       # Main application component
│   │   └── index.tsx     # Application entry point
│   ├── public/           # Static assets
│   └── package.json      # npm dependencies
└── README.md             # This file
```

## Security Features

- **CORS Protection**: Configured CORS middleware
- **Rate Limiting**: 100 requests per minute per IP
- **Security Headers**: HSTS, CSP, X-Frame-Options, etc.
- **File Validation**: Magic number validation for image formats
- **Automatic Cleanup**: Temporary files deleted after 1 hour
- **HTTPS Enforcement**: Security headers for production deployment

## Development

### Adding New Features

1. Update requirements in `.kiro/specs/virtual-fitcheck/requirements.md`
2. Update design in `.kiro/specs/virtual-fitcheck/design.md`
3. Add tasks to `.kiro/specs/virtual-fitcheck/tasks.md`
4. Implement features with property-based tests
5. Run tests to ensure correctness

### Property-Based Testing

This project uses property-based testing to ensure correctness:

- **Backend**: gopter library for Go
- **Frontend**: fast-check library for TypeScript

Each correctness property from the design document is implemented as a property-based test that runs 100+ iterations with randomly generated inputs.

## Troubleshooting

### Backend won't start

- Check that port 8080 is not in use
- Verify AI service credentials are correct
- Ensure storage path is writable

### Frontend can't connect to backend

- Verify backend is running on the correct port
- Check `REACT_APP_API_URL` in frontend `.env`
- Ensure CORS is properly configured

### Upload fails

- Check file size (must be under 10MB)
- Verify file format (JPEG, PNG, or WebP only)
- Ensure storage path exists and is writable

### Processing takes too long

- Check AI service status and API key
- Verify network connectivity to AI service
- Check backend logs for errors

## License

MIT

## Contributing

Contributions are welcome! Please follow the spec-driven development process outlined in the `.kiro/specs` directory.

## Support

For issues and questions, please open an issue on the project repository.
