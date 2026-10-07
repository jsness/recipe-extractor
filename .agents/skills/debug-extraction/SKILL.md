---
name: debug-extraction
description: Diagnose failed, stuck, or incomplete recipe URL extractions in this repository; trace scraping, JSON-LD, model fallback, persistence, and archive retry, and reproduce code defects with offline regression tests.
---

# Debug extraction

Identify the first failing stage using evidence. For a diagnosis request, report the cause and next action; when a fix is requested, make a targeted change and add a reproducible regression test.

## Start with the failing extraction

- Read the root `AGENTS.md`. Paths below are relative to the repository root; inspect current source before relying on the behavior described here.
- Collect the source URL, expected versus observed result, extraction ID/profile ID if available, status/error, and relevant worker logs. Ask only for missing evidence needed to proceed; source and existing fixtures can often narrow the problem first.
- Check the running environment and effective `EXTRACTOR`, `LLM_ONLY_EXTRACTION`, model, base URL, and timeout settings. Inspect `server/internal/config/config.go` and `docs/environment-variables.md`. Record whether a required key is present without printing keys or credential-bearing database URLs.
- Prefer existing logs and read-only status requests before submitting another extraction. An API submission writes data and can call a paid model or queue linked recipes; use an isolated dev profile/database for reproduction within the requested scope. Do not reset the user's database or force-update queue rows as a diagnostic shortcut.

Useful commands (from root unless stated otherwise):

```powershell
rg -n 'processing extraction|extraction failed|json-ld parse|mark done|claim queued' server
docker compose logs --tail 200 app
```

Container logs apply only to a Compose-run backend; use the Go process output for script-based development. See `AGENTS.md` for actual ports; do not assume the live app uses the dev port.

Read-only API checks, after replacing the example values:

```powershell
$baseUrl = 'http://localhost:8081'
$profileId = 'replace-with-profile-id'
$extractionId = 'replace-with-extraction-id'
Invoke-RestMethod "$baseUrl/healthz"
Invoke-RestMethod "$baseUrl/api/v1/recipe-extractions/$extractionId" -Headers @{ 'X-Profile-Id' = $profileId }
```

`/healthz` confirms the HTTP handler responds, not that the database or worker is healthy. Extraction reads require the matching `X-Profile-Id`; a 404 may indicate the wrong profile. `GET /api/v1/profiles` can identify existing profiles.

## Locate the first failing stage

| Evidence or symptom | Inspect | What to distinguish |
| --- | --- | --- |
| Submission rejected; no extraction ID | `server/internal/api/api.go`, `server/core/app.go` | POST `/api/v1/recipes` normally returns 202. Missing profile header returns 400; an existing queued/extracting or completed extraction with a recipe produces 409. Failed extractions can be retried. |
| Remains `queued` | `server/cmd/server/main.go`, `server/worker/worker.go`, `server/store/extractions.go` | Worker startup, claim errors, correct database, and older jobs. Claims use a transaction with `FOR UPDATE SKIP LOCKED`. |
| Remains `extracting` | `server/worker/worker.go`, `server/store/extractions.go` | Claim commits before fetching. Check process interruption, network/model timeout, and status-write errors. Inspect recovery behavior rather than assuming a restart requeues claimed work. |
| Error begins `scrape:` | `server/scraper/scraper.go`, `server/scraper/robots.go` | robots rules, transport failures, redirects, HTTP status, challenge classification, and body limits. A page visible in a browser may still fail the scraper. |
| Error begins `normalize recipe:` or fields are missing | `server/extractor/jsonld.go`, `normalize.go`, provider files | Scraper inputs, Recipe-node selection, required fields, ingredient grouping, fallback configuration, model HTTP/JSON errors, and normalization. |
| Error begins `store recipe:` or `mark done:` | `server/store/recipes.go`, `extractions.go`, migrations | Database errors and schema; whether the recipe was saved before the final status write failed. Preserve profile scoping. |
| Extraction is done but UI is wrong | `server/internal/api/responses.go`, `web/src/RecipeApp.tsx`, `web/src/components/ExtractionCard.tsx` | Compare saved recipe/API data with polling, active profile, error display, and dismissal state before changing extraction logic. |

Pipeline details that affect diagnosis:

- Scraper reads at most 2 MiB of HTML and caps text at 20,000 bytes; model prompt construction applies further limits. Inspect whether the relevant content reaches `extractor.Input` (`JSONLD`, `Text`, `Links`, `Ingredients`, `SourceURL`).
- Normal mode tries JSON-LD blocks first and invokes a configured model only when structured parsing fails. `LLM_ONLY_EXTRACTION` selects the model directly. Missing cloud credentials can leave normal mode with JSON-LD only; local extraction does not require a key by default.
- Check root versus `@graph` nodes, `@type`, instruction shapes, title, ingredients, and instructions against the actual parser. Do not assume all valid schema.org representations are supported. Inspect `reconcileRecipeWithStructuredData` in `openai.go` when model output differs from the saved result; `local.go` shares the chat-completion implementation.
- Linked-recipe queue/relationship errors are logged separately and can occur while the primary extraction succeeds. Diagnose missing components independently.

## Diagnose archive retry only when relevant

Read `server/internal/api/responses.go`, `server/scraper/scraper.go`, `server/wayback/wayback.go`, and the archive handler in `web/src/RecipeApp.tsx`.

`can_try_archived_source` is set only for a failed extraction whose error matches `SupportsArchivedFallback` and whose source is not already on `web.archive.org`. A generic 403, robots denial, or model failure does not automatically qualify. Preserve the relationship between the scraper's blocked-access message and this flag.

The UI calls `GET /api/v1/archived-snapshot?url=...`, then submits the returned archive URL as a new extraction. The worker does not automatically retry the original job against Wayback. Distinguish no available snapshot, lookup failure, archived-page scrape failure, and archived-page parsing failure. Keep both the original and archive URL in diagnostic evidence.

## Turn a code defect into an offline regression

- Preserve the smallest input that demonstrates the defect: JSON-LD block, HTML fragment, provider response, or extraction state. Use inline fixtures for small cases and the affected package's `testdata/` for larger ones. Remove cookies, credentials, personal data, and irrelevant page content; retain the exact failing structure.
- Place the test at the earliest responsible layer. Follow existing table-driven cases in `scraper_test.go`, `jsonld_test.go`, `normalize_test.go`, `local_test.go`, `worker_test.go`, `core/app_test.go`, or API response tests.
- For HTTP behavior, use `httptest.NewServer` with explicit status, headers, and body; scraper fetch tests should also serve `/robots.txt`. Mock model endpoints/fallbacks instead of depending on live websites, Wayback, credentials, or paid models. If a database reproduction is necessary, use the isolated dev database and disclose that requirement.
- Assert the user's missing fields, expected error/status, or fallback behavior. For a requested fix, demonstrate that the regression fails before the change and passes after it. Do not alter prompts, switch providers, or raise timeouts unless evidence identifies that as the cause.

Run from `server/`:

```text
go test ./<affected-package> -run <RegressionTestName> -count=1
go test ./...
```

Format changed Go files with `gofmt`. If the fix touches UI polling or error display, run frontend lint, TypeScript checks, and build as documented in `AGENTS.md`, then verify the relevant UI state.

## Report

State the failing stage, supporting evidence, root cause or remaining uncertainty, and the fix or next action. Link any fixture/test and list checks actually run. Distinguish an offline reproduction from a verified live retry; do not claim a source URL now works without checking it.
