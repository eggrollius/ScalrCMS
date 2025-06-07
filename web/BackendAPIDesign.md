# ScalrCMS Video Upload REST API Specification (Hybrid Upload Flow)

## Overview

This document outlines the REST API endpoints for the hybrid video upload flow in ScalrCMS, where an empty video record is created before upload begins. This allows concurrent video uploading and metadata submission, with backend-controlled S3 keys and secure pre-signed URLs.

---

## API Endpoints

| Method | Path                     | Purpose                                    | Request Body / Params                            | Response                              |
|--------|--------------------------|--------------------------------------------|-------------------------------------------------|-------------------------------------|
| POST   | `/api/videos/init`       | Create empty video record and generate pre-signed upload URL | JSON: `{}` (optional user/session info)          | JSON: `{ videoId: string, uploadUrl: string }` |
| PUT    | `/api/videos/:id/metadata` | Submit metadata for existing video record    | JSON: `{ title, description, tags, visibility }` | JSON: `{ message: "Metadata accepted" }`         |
| GET    | `/api/videos/:id/status` | Get processing status and resolution info     | Path param: video ID                              | JSON: see example below              |
| GET    | `/api/videos/:id`        | Get video metadata & playback URLs            | Path param: video ID                              | JSON: see example below              |

---

## API Flow

1. **Initialize Upload:**  
   Client calls `POST /api/videos/init` when visiting the upload page (or selecting a file).  
   → Backend creates an empty video record with a unique GUID and S3 key, generates a pre-signed upload URL scoped to that key, and returns both to the client.

2. **Upload Video:**  
   Client uploads the video file directly to S3 using the provided pre-signed URL.

3. **Submit Metadata:**  
   Client calls `PUT /api/videos/:id/metadata` with the video metadata using the `videoId` from initialization.

4. **Processing & Status:**  
   Backend stores metadata, triggers video processing, and client polls `GET /api/videos/:id/status` for processing progress.

5. **Retrieve Video Details:**  
   Client fetches video info and playback URLs via `GET /api/videos/:id`.

6. **Cleanup:**  
   Backend runs periodic cleanup jobs to delete empty or abandoned video records and orphaned uploads.

---

## Example Requests and Responses

### 1. Initialize Upload

**Request:**  
