# Trailarr

Trailarr is a self-hosted web application for managing and downloading movie and TV show extras (trailers, featurettes, etc.) for media libraries managed by Radarr and Sonarr. It features a Go backend API and a modern React frontend.

## Features

- **Automatic and manual download of extras** (trailers, featurettes, etc.) for movies and TV series
- **Integration with Radarr and Sonarr** for seamless media management
- **Web-based UI** built with React and Vite
- **Background tasks** for scheduled downloads and status updates
- **History and status tracking** for extras
- **Customizable settings** for Radarr, Sonarr, and extra types
- **Media file browser** for server-side directory picking
- **Poster/banner serving** and static asset hosting

## Architecture

- **Backend:** Go (Gin framework)
  - REST API for extras management, settings, and status
  - Serves static files and React SPA
  - Handles background tasks and sync timings
- **Frontend:** React + Vite
  - Modern, responsive UI for managing extras and settings
  - Communicates with backend via REST API

## Project Structure

- `cmd/trailarr/` — Main Go application entrypoint
- `internal/` — Backend logic, API handlers, background tasks, settings, and integrations
- `web/` — React frontend (Vite project)
- `bin/` — Compiled Go binaries
- `mediacover/`, `posters/` — Media assets
- `deployments/`, `scripts/`, `test/` — Deployment, scripts, and tests

## API Endpoints (selected)

- `GET /api/health` — Health check
- `GET /api/movies`, `GET /api/series` — List movies/series
- `GET /api/movies/:id/extras`, `GET /api/series/:id/extras` — List extras for a movie/series
- `POST /api/extras/download` — Download an extra
- `DELETE /api/extras` — Delete an extra
- `GET /api/history` — Download history
- `GET/POST /api/settings/*` — Get/set settings for Radarr, Sonarr, general, and extra types
- `GET /api/files/list` — Server-side file browser

## Build & Run

### Prerequisites
- Go 1.25+
- Node.js 20+
- Make
- Docker (optional, for containerized deployment)

### Local Build

1. **Build backend:**
   ```sh
   make build
   ```
2. **Run backend:**
   ```sh
   make run
   ```
3. **Build frontend:**
   ```sh
   cd web && npm install && npm run build
   ```
4. **Access UI:**
   Open [http://localhost:8080](http://localhost:8080)

### Docker

Build and run the container:
```sh
docker build -t trailarr:latest .
docker run -p 8080:8080 trailarr:latest
```

## Configuration
- Settings for Radarr, Sonarr, and extras are managed via the web UI.
- Sync timings and other advanced settings are loaded from config files (see `internal/`).

## License
MIT

---
*This README was generated automatically. Please update with project-specific details as needed.*
