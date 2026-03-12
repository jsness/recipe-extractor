# Environment Variables

Copy `.env.example` to `.env` and set values as needed.

| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Address the Go server listens on |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable` | PostgreSQL connection string |
| `EXTRACTOR` | `openai` | LLM backend to use when the app needs an LLM: `anthropic` or `openai` |
| `LLM_ONLY_EXTRACTION` | `false` | When `true`, skip JSON-LD and always use the configured LLM extractor; requires a valid API key for the selected provider |
| `ANTHROPIC_API_KEY` | _(optional)_ | Anthropic API key; if unset, Anthropic cannot be used for fallback or LLM-only extraction |
| `ANTHROPIC_MODEL` | `claude-sonnet-4-6` | Anthropic model ID |
| `ANTHROPIC_TIMEOUT_SECONDS` | `45` | Request timeout for Anthropic calls |
| `OPENAI_API_KEY` | _(optional)_ | OpenAI API key; if unset, OpenAI cannot be used for fallback or LLM-only extraction |
| `OPENAI_MODEL` | `gpt-5-mini` | OpenAI model ID |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | OpenAI-compatible base URL |
| `OPENAI_PROJECT_ID` | _(optional)_ | OpenAI project ID |
| `OPENAI_ORGANIZATION_ID` | _(optional)_ | OpenAI organization ID |
| `OPENAI_TIMEOUT_SECONDS` | `45` | Request timeout for OpenAI calls |

> **Note:** Postgres is mapped to host port `5433` (not the default `5432`) to avoid clashing with other local Postgres instances.
>
> With `LLM_ONLY_EXTRACTION=false`, extraction remains JSON-LD first with optional LLM fallback. With `LLM_ONLY_EXTRACTION=true`, the server fails to start unless the selected LLM provider is configured.
