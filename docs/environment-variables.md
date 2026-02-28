# Environment Variables

Copy `.env.example` to `.env` and set values as needed.

| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Address the Go server listens on |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5433/recipes?sslmode=disable` | PostgreSQL connection string |
| `EXTRACTOR` | `anthropic` | LLM backend to use: `anthropic` or `openai` |
| `ANTHROPIC_API_KEY` | _(required if `EXTRACTOR=anthropic`)_ | Anthropic API key |
| `ANTHROPIC_MODEL` | `claude-sonnet-4-6` | Anthropic model ID |
| `ANTHROPIC_TIMEOUT_SECONDS` | `45` | Request timeout for Anthropic calls |
| `OPENAI_API_KEY` | _(required if `EXTRACTOR=openai`)_ | OpenAI API key |
| `OPENAI_MODEL` | `gpt-5-mini` | OpenAI model ID |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | OpenAI-compatible base URL |
| `OPENAI_PROJECT_ID` | _(optional)_ | OpenAI project ID |
| `OPENAI_ORGANIZATION_ID` | _(optional)_ | OpenAI organization ID |
| `OPENAI_TIMEOUT_SECONDS` | `45` | Request timeout for OpenAI calls |

> **Note:** Postgres is mapped to host port `5433` (not the default `5432`) to avoid clashing with other local Postgres instances.
