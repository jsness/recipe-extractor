# Recipe Extractor

[![Publish Docker image](https://github.com/jsness/recipe-extractor/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/jsness/recipe-extractor/actions/workflows/docker-publish.yml) ![GitHub Tag](https://img.shields.io/github/v/tag/jsness/recipe-extractor) ![GitHub License](https://img.shields.io/github/license/jsness/recipe-extractor)

A web app that scrapes a recipe URL, extracts it, and saves it to a searchable library. Paste a URL, click Extract, and the backend fetches the page, parses structured data (JSON-LD), and falls back to an AI model when needed to produce a clean structured recipe. You can also configure it to skip JSON-LD entirely and run LLM-only extraction.

Most recipe websites bury ingredients and instructions inside walls of ads, pop-ups, and life stories. Recipe Extractor strips all of that away. You get just the recipe, stored in a consistent structure you actually own, with no account required and no dependency on the original site staying online.

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

### Local development

For hot reload, copy `.env.example` to `.env`, optionally set `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`, then run:

**Linux/macOS**
```bash
./scripts/dev.sh
```

**Windows**
```powershell
.\scripts\dev.ps1
```

The Go server runs on `:8080` and the Vite dev server runs on `:5173` (proxies `/api/*` to the backend automatically).

Both dev scripts also set `FRONTEND_DEV_PROXY_URL=http://localhost:5173` for the local Go server process, so you can open the app at **http://localhost:8080** while still using Vite hot reload behind the scenes.

## Remote Access

If you want to use Recipe Extractor from other devices, put it behind an access layer instead of exposing it directly. [Tailscale](https://tailscale.com) is a good fit for private access within your tailnet, and [Cloudflare Access](https://www.cloudflare.com/products/zero-trust/access/) works well if you want a public URL with login-gated access.

## Documentation

| Guide | Description |
|---|---|
| [API Reference](docs/api-reference.md) | All endpoints, request/response shapes, and curl examples |
| [Environment Variables](docs/environment-variables.md) | Full reference for every config variable and its default |
| [Go Module Integration](docs/go-module-integration.md) | Import `recipe-extractor` into another Go app and mount or call it directly |
| [Utility Scripts](docs/utility-scripts.md) | PowerShell helper scripts for local development |
