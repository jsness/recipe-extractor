# Extraction reliability

Extraction runs in Go and keeps the existing JSON-LD-first behavior unless
`LLM_ONLY_EXTRACTION=true`. The following recovery steps run automatically within
an extraction; they do not require a new service or database migration.

## Network retries

Page requests, robots.txt requests, Wayback availability lookup, and Wayback CDX
snapshot lookup each allow at most three GET attempts. Retries cover temporary
transport failures, interrupted response bodies, and HTTP 408, 429, 500, 502,
503, and 504. Other HTTP errors return immediately. Detected challenge pages
also return immediately, including challenges returned with HTTP 200.

Backoff starts at 500 milliseconds, doubles, and adds jitter. A valid
`Retry-After` header (seconds or HTTP date) determines the wait instead. If the
requested wait exceeds 30 seconds, the request stops rather than retrying before
the server permits. Caller cancellation interrupts requests and backoff waits.

The HTTP client timeout applies to each attempt. Page and robots.txt requests
use the selected extractor's timeout, or 30 seconds when no model is configured.
Wayback API requests use 10 seconds. Retrying can therefore increase the total
time an extraction spends fetching a page.

Temporary robots.txt failures stop the current fetch and are not cached. A later
extraction checks again. A robots.txt denial stops extraction. Successful rules
and missing robots.txt responses retain the existing in-memory cache behavior.

Responses over the limits (2 MiB for pages, 512 KiB for robots.txt, 1 MiB for
Wayback API responses) fail explicitly rather than silently parsing a truncated
response. Extracted page text remains capped at 20,000 bytes.

## Structured recipe parsing

JSON-LD can contain a recipe object, a top-level array, or an `@graph` collection,
including graphs inside arrays. An incomplete Recipe node does not hide a later
valid node in the same collection. The first usable recipe is normalized and
validated as before. Existing HTML ingredient-group reconciliation is preserved.

The exported `extractor.ErrInvalidRecipe` identifies missing or malformed recipe
content. This lets archive recovery distinguish content failures from model
authentication, rate-limit, or other provider failures.

## Archive recovery

The **Try archived version** action still looks up a snapshot and submits its
archive URL as a separate extraction. The worker attempts that capture first.
If fetching or recipe-content normalization fails, it searches the Wayback CDX
index for at most three recent HTML captures with recorded HTTP status 200,
collapsing adjacent duplicate content digests. Index status does not guarantee
that a capture contains a usable recipe; each candidate is still fetched and
validated.

The worker tries at most two alternatives, skipping the initial capture,
duplicate timestamps for the same source, and captures of a different URL.
Older HTTP captures of an HTTPS source are eligible when host, path, and query
match. Alternate recovery has a two-minute deadline, including candidate lookup,
fetches, waits, and model calls. Robots failures and provider failures stop recovery.

On success, the saved recipe's `source_url` points to the capture that supplied
the recipe. The extraction status response continues to identify the submitted
URL and links to the resulting recipe via `recipe_id`.

## Diagnostics and testing

The Docker builder uses Go 1.27.1, matching the toolchain used to verify native
and container extraction. Go's HTTP/TLS behavior can change between toolchains,
and a site may challenge one build while accepting another. When updating the
builder, verify fetching from the built runtime image as well as running offline
tests. A successful native extraction alone does not verify the Docker build.

Worker logs include extraction IDs and normalization duration. HTTP logs include
stage, host, attempt, status, duration, and retry wait. Archive recovery logs
include the selected snapshot URL and timestamp, each failure, and success.
Request and response bodies are not added to these diagnostics. Existing local
model traffic logging retains its separate behavior.

Regression tests use mock HTTP servers and small JSON-LD fixtures. They exercise
retry exhaustion, cancellation, Retry-After, HTTP 200 challenges, transient
robots failures, recipe collections, and archive candidate selection without
calling live websites, Wayback, or paid models.
