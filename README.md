# Recipe Extractor

[![Publish Docker image](https://github.com/jsness/recipe-extractor/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/jsness/recipe-extractor/actions/workflows/docker-publish.yml) ![GitHub Tag](https://img.shields.io/github/v/tag/jsness/recipe-extractor) ![GitHub License](https://img.shields.io/github/license/jsness/recipe-extractor)

A web app that scrapes a recipe URL, extracts it, and saves it to a searchable library. Paste a URL, click Extract, and the backend fetches the page, parses structured data (JSON-LD), and falls back to an AI model when needed to produce a clean structured recipe. You can also configure it to skip JSON-LD entirely and run LLM-only extraction.

Most recipe websites bury ingredients and instructions inside walls of ads, pop-ups, and life stories. Recipe Extractor strips all of that away. You get just the recipe, stored in a consistent structure you actually own, with no account required and no dependency on the original site staying online.

## How It Works

1. **Scraper** - fetches the target page and pulls out JSON-LD structured data and visible plaintext
2. **Extractor** - parses Schema.org JSON-LD structured data directly by default; falls back to an LLM (Anthropic or OpenAI) when JSON-LD is absent or incomplete, or can be configured to run LLM-only
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

**Requires:** [Docker](https://docs.docker.com/get-docker/). An [Anthropic](https://console.anthropic.com) or [OpenAI](https://platform.openai.com) API key is optional by default - recipes are extracted from structured data (JSON-LD) automatically, and the LLM is only used as a fallback unless `LLM_ONLY_EXTRACTION=true`.

### No-clone install (recommended)

```bash
mkdir recipe-extractor && cd recipe-extractor
curl -o compose.yml https://raw.githubusercontent.com/jsness/recipe-extractor/main/compose.yml
curl -o .env https://raw.githubusercontent.com/jsness/recipe-extractor/main/.env.example
nano .env   # optionally set ANTHROPIC_API_KEY or OPENAI_API_KEY; set LLM_ONLY_EXTRACTION=true to force LLM-only mode
docker compose up -d
```

Open **http://localhost:8080**. Data persists in a Docker volume across restarts. Port can be changed in `compose.yml`.

### Build from source

```bash
git clone https://github.com/jsness/recipe-extractor
cd recipe-extractor
cp .env.example .env   # optionally set ANTHROPIC_API_KEY or OPENAI_API_KEY; set LLM_ONLY_EXTRACTION=true to force LLM-only mode
docker compose build && docker compose up -d
```

Open **http://localhost:8080**.

### Local development (Windows)

If you want to work on the code with hot-reloading:

```powershell
cp .env.example .env   # optionally set ANTHROPIC_API_KEY or OPENAI_API_KEY; set LLM_ONLY_EXTRACTION=true to force LLM-only mode
.\scripts\dev.ps1
```

The Go server runs on `:8080` and the Vite dev server runs on `:5173` (proxies `/api/*` to the backend automatically).

`.\scripts\dev.ps1` also sets `FRONTEND_DEV_PROXY_URL=http://localhost:5173` for the local Go server process, so you can open the app at **http://localhost:8080** while still using Vite hot reload behind the scenes.

## Using an Existing PostgreSQL Instance

If you already have a PostgreSQL instance running, set `DATABASE_URL` in your `.env` to point at it and start only the app service - skipping the bundled postgres container entirely:

```bash
# .env
DATABASE_URL=postgres://user:password@your-host:5432/recipes?sslmode=disable
```

```bash
docker compose up -d app
```

The app will run migrations automatically on startup, so no manual schema setup is needed.

## Remote Access with Tailscale

To access your recipe library from your phone, laptop, or anywhere else without exposing anything to the public internet, [Tailscale](https://tailscale.com) is the easiest option:

1. Install Tailscale on the machine running the stack and sign in
2. `docker compose up -d` as normal
3. Open `http://<tailscale-ip>:8080` from any device on your tailnet

Your Tailscale IP is shown in the Tailscale app or via `tailscale ip -4`. No port forwarding or firewall rules needed.

## Prerequisites (local dev only)

- [Go 1.24+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/) and npm
- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- An API key for either Anthropic or OpenAI _(optional by default; required if `LLM_ONLY_EXTRACTION=true`)_

## Documentation

| Guide | Description |
|---|---|
| [API Reference](docs/api-reference.md) | All endpoints, request/response shapes, and curl examples |
| [Environment Variables](docs/environment-variables.md) | Full reference for every config variable and its default |
| [Utility Scripts](docs/utility-scripts.md) | PowerShell helper scripts for local development |
