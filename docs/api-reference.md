# API Reference

All endpoints are under `/api/v1`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/profiles` | List available profiles. |
| `POST` | `/api/v1/profiles` | Create a profile. Body: `{"name": "Weeknight"}`. |
| `POST` | `/api/v1/recipes` | Submit a URL for extraction. Body: `{"url": "https://..."}`. Returns `extraction_id` and initial `status`. |
| `GET` | `/api/v1/recipe-extractions/{id}` | Poll extraction status. Status values: `queued`, `extracting`, `done`, `failed`. |
| `GET` | `/api/v1/recipes` | List saved recipes for the active profile. Requires header `X-Profile-Id`. |
| `GET` | `/api/v1/recipes/{id}` | Get full recipe details for the active profile. Requires header `X-Profile-Id`. |
| `DELETE` | `/api/v1/recipes/{id}` | Delete a saved recipe in the active profile. Requires header `X-Profile-Id`. Returns `204` on success and `404` if the recipe does not exist. |
| `PATCH` | `/api/v1/recipes/{id}/reminder` | Save or remove a user reminder. Requires `X-Profile-Id`. Body: `{"reminder":"Use less salt next time."}`. Returns updated recipe details. |
| `GET` | `/healthz` | Health check. |

All recipe and extraction endpoints require the `X-Profile-Id` header.

## Recipe reminders

Migration `006_recipe_reminders.sql` renames the former notice column to
`reminder`, preserving previously saved text. Restart the backend after updating
so it applies the migration and exposes the renamed endpoint.

Recipe details include an optional `reminder` string, separate from extracted `notes`.
Reminders are written by users and are preserved during extraction updates. A reminder
appears above Ingredients in the UI and in printed recipes.

Use `PATCH /api/v1/recipes/{id}/reminder` with `Content-Type: application/json` and
the active profile's `X-Profile-Id` header to add or edit a reminder:

```json
{"reminder":"Use less salt next time.\nBake five extra minutes."}
```

Send `{"reminder":""}` to remove it. Leading/trailing whitespace is trimmed;
whitespace-only text also removes the reminder. Internal line breaks are preserved.
Success returns `200` with full recipe details; `reminder` is omitted after removal.
Missing/null/non-string reminders, unknown fields, malformed JSON, or invalid IDs
return `400`. Missing recipes and recipes outside the requested profile return
`404`; unexpected persistence errors return `500`. Only the reminder is updated.

For database integration tests, set `RECIPE_TEST_DATABASE_URL` to a disposable
PostgreSQL database before running `go test ./...` from `server/`. The reminder test
applies migrations and creates temporary fixtures. Without that variable, the
database integration test is skipped.

## Example: Create a Profile

```bash
curl -X POST http://localhost:8080/api/v1/profiles \
  -H "Content-Type: application/json" \
  -d '{"name": "Weeknight"}'
```

```json
{
  "id": "a1b2c3d4-...",
  "name": "Weeknight",
  "created_at": "2026-03-28T12:00:00Z"
}
```

## Example: Submit a URL

```bash
curl -X POST http://localhost:8080/api/v1/recipes \
  -H "Content-Type: application/json" \
  -H "X-Profile-Id: a1b2c3d4-..." \
  -d '{"url": "https://example.com/chocolate-chip-cookies"}'
```

```json
{
  "extraction_id": "a1b2c3d4-...",
  "status": "queued"
}
```

## Example: Poll Status

```bash
curl http://localhost:8080/api/v1/recipe-extractions/a1b2c3d4-... \
  -H "X-Profile-Id: a1b2c3d4-..."
```

```json
{
  "id": "a1b2c3d4-...",
  "source_url": "https://example.com/chocolate-chip-cookies",
  "status": "done",
  "recipe_id": "e5f6g7h8-..."
}
```
