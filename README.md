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

## Quickstart

**Requires:** [Docker](https://docs.docker.com/get-docker/) and an
[Anthropic](https://console.anthropic.com) or [OpenAI](https://platform.openai.com) API key.

### No-clone install (recommended)

```bash
mkdir recipe-extractor && cd recipe-extractor
curl -o compose.yml https://raw.githubusercontent.com/jsness/recipe-extractor/main/compose.yml
curl -o .env https://raw.githubusercontent.com/jsness/recipe-extractor/main/.env.example
nano .env   # set ANTHROPIC_API_KEY or OPENAI_API_KEY
docker compose up -d
```

Open **http://localhost:8080**. Data persists in a Docker volume across restarts.

### Build from source

```bash
git clone https://github.com/jsness/recipe-extractor
cd recipe-extractor
cp .env.example .env   # set ANTHROPIC_API_KEY or OPENAI_API_KEY
docker compose build && docker compose up -d
```

Open **http://localhost:8080**.

### Local development (Windows)

If you want to work on the code with hot-reloading:

```powershell
cp .env.example .env   # set ANTHROPIC_API_KEY or OPENAI_API_KEY
.\scripts\dev.ps1
```

The Go server runs on `:8080` and the Vite dev server runs on `:5173` (proxies `/api/*` to the backend automatically).

## Prerequisites (local dev only)

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/) and npm
- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- An API key for either Anthropic or OpenAI

## Documentation

| Guide | Description |
|---|---|
| [API Reference](docs/api-reference.md) | All endpoints, request/response shapes, and curl examples |
| [Environment Variables](docs/environment-variables.md) | Full reference for every config variable and its default |
| [Utility Scripts](docs/utility-scripts.md) | PowerShell helper scripts for local development |
