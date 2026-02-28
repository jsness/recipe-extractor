# Recipe Extractor

A web app that scrapes a recipe URL, normalizes it with an LLM, and saves it to a searchable library. Paste a URL, click Extract, and the backend fetches the page, sends the content to an AI model, and stores a clean structured recipe.

Most recipe websites bury ingredients and instructions inside walls of ads, pop-ups, and life stories. Recipe Extractor strips all of that away. You get just the recipe, stored in a consistent structure you actually own, with no account required and no dependency on the original site staying online.

## How It Works

1. **Scraper** - fetches the target page and pulls out JSON-LD structured data and visible plaintext
2. **Extractor** - sends the scraped content to an LLM (Anthropic or OpenAI) which returns a normalized JSON recipe
3. **Worker** - a background goroutine polls the database for queued extraction jobs and processes them asynchronously
4. **API** - a Go/chi REST API serves recipe data and exposes extraction status
5. **Frontend** - a React + Mantine SPA polls for extraction status and displays the recipe library

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24, chi router |
| Database | PostgreSQL 16 |
| LLM | Anthropic Claude or OpenAI (configurable) |
| Frontend | React 18, TypeScript, Vite, Mantine 8 |
| Dev DB | Docker Compose |

## Prerequisites

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/) and npm
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (for the local Postgres container)
- An API key for either Anthropic or OpenAI

## Quickstart (Windows)

```powershell
cd C:\dev\recipe-extractor

# 1. Copy environment template and fill in your API key
copy .env.example .env
notepad .env

# 2. Start everything (Postgres container + Go server + Vite dev server)
.\scripts\dev.ps1
```

Then open `http://localhost:5173` in your browser.

The `dev.ps1` script:
- Starts the Docker Compose Postgres container if it isn't already running
- Runs the Go server (`server/`) in the background
- Runs the Vite dev server (`web/`) in the foreground

## Manual Setup

If you prefer to run each piece yourself:

```powershell
# 1. Start Postgres
docker compose up -d

# 2. Start the Go backend (from repo root)
cd server
go run ./cmd/server

# 3. Start the frontend dev server (separate terminal)
cd web
npm install
npm run dev
```

Backend listens on `:8080`. Frontend dev server listens on `:5173` and proxies `/api/*` requests to the backend automatically.

## Building for Production

```powershell
# Build the frontend
cd web
npm run build
# Output goes to web/dist/

# Build the Go binary
cd server
go build -o recipe-extractor.exe ./cmd/server
```

Serve `web/dist/` as static files from the Go server or a separate static host, and point the frontend's API calls at the Go binary.

## Documentation

| Guide | Description |
|---|---|
| [API Reference](docs/api-reference.md) | All endpoints, request/response shapes, and curl examples |
| [Environment Variables](docs/environment-variables.md) | Full reference for every config variable and its default |
| [Utility Scripts](docs/utility-scripts.md) | PowerShell helper scripts for local development |
