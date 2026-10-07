# Agent guide

## Commands

Run commands in the indicated directory. Requires Go 1.24+, Node.js 20.19+ (or 22.12+), and Docker Compose for local PostgreSQL. On Windows, use `npm.cmd`/`npx.cmd` if PowerShell blocks the wrappers.

| Command | Directory | Purpose |
| --- | --- | --- |
| `npm ci` | `web/` | Install locked frontend dependencies |
| `npm run lint` | `web/` | ESLint checks |
| `npx tsc --noEmit` | `web/` | Strict TypeScript checks; Vite build does not type-check |
| `npm run build` | `web/` | Build frontend into `web/dist/` |
| `go test ./...` | `server/` | Run Go tests |
| `go build -o recipe-extractor ./cmd/server` | `server/` | Build server binary |
| `gofmt -w <changed.go files>` | `server/` | Format changed Go files |
| `git diff --check` | root | Check patch whitespace |
| `.\scripts\check-bom.ps1` | root | Check tracked files for UTF-8 BOMs |
| `docker compose build app` | root | Build full application with embedded frontend |
| `.\scripts\dev.ps1` / `./scripts/dev.sh` | root | Start isolated dev database, Go server, and Vite |

For local development, copy `.env.example` to `.env` if absent and install frontend dependencies first. The shell script requires **zsh**. Dev scripts default to Go `:8081`, Vite `:5174`, PostgreSQL `:5434`; open `http://localhost:8081`. See scripts for `DEV_*` overrides. Plain Vite defaults to `:5173` and proxies API requests to `:8080`.

A direct Go build uses the checked-in frontend placeholder. For a usable embedded UI, build via Docker, which copies `web/dist/` into `server/internal/frontend/dist/`. There is no frontend test runner configured.

## Purpose and stack

Extract recipe URLs into a searchable, profile-scoped library using JSON-LD, optional Anthropic/OpenAI/local model extraction, and Wayback retry support. Stack: Go with chi, pgx, PostgreSQL 16, embedded SQL migrations; React 18, strict TypeScript, Vite 7, Mantine 8; Docker multi-stage build.

## Layout

- `server/cmd/server/`: executable entry point.
- `server/core/`, `server/httpapi/`: reusable application and public HTTP integration.
- `server/internal/`: API handlers/responses, configuration, database setup, embedded frontend.
- `server/store/`: persistence and data types; `server/migrations/`: numbered SQL migrations.
- `server/scraper/`, `extractor/`, `worker/`, `wayback/`: fetching, extraction, queue processing, archive lookup.
- `web/src/`: app state, shared `types.ts`, `components/`, `utils/`, `icons/`.
- `docs/`: API, environment, module integration, script references.
- `scripts/`: development, BOM check, destructive database reset utility.
- `.github/workflows/docker-publish.yml`: publishes images for qualifying main/tag pushes; does not run tests.

## Conventions

- Use `gofmt`; keep persistence in `store`, application logic in `core`, and HTTP serialization in `internal/api`. Preserve profile scoping.
- Follow `web/eslint.config.js`: two-space indentation, multiline trailing commas, spaced object braces, one JSX prop per line in multiline tags. Match existing double quotes and semicolons; use PascalCase components and typed props.
- Keep frontend API types aligned with Go responses. Add numbered migrations for schema changes and update relevant docs/config examples for contract changes.
- Save text as UTF-8 without BOM. Keep secrets in ignored `.env`; do not commit generated assets or binaries.

## Pull requests

- Keep changes focused; preserve unrelated local edits.
- Run relevant checks above before submitting; add Go regression tests for changed behavior and manually verify UI changes.
- Describe the problem, resulting behavior, and validation; disclose checks not run and migration/config requirements. Include screenshots for visible UI changes.
- No repository PR template or enforced commit-message format exists; use concise, descriptive titles.
