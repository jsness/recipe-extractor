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
| `GET` | `/healthz` | Health check. |

All recipe and extraction endpoints require the `X-Profile-Id` header.

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
