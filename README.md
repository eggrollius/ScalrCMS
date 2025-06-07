# ScalrCMS Monorepo

This project contains:
- **Backend**: Golang API using Gin and GORM (root directory)
- **Frontend**: React SPA using Vite + TypeScript (`/web` directory)

## Getting Started

### Backend (Golang)
1. Install Go (https://golang.org/dl/)
2. Run the API:
   ```bash
   go run main.go
   ```

### Frontend (React)
1. Navigate to the `web` directory:
   ```bash
   cd web
   ```
2. Start the development server:
   ```bash
   npm run dev
   ```

## Project Structure
- `/main.go` — Gin API entry point
- `/web/` — React SPA frontend

---

## Development Notes
- Use RESTful conventions for API endpoints.
- Use functional React components and hooks.