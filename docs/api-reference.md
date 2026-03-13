# API Reference

All endpoints are under `/api/v1`.

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/recipes` | Submit a URL for extraction. Body: `{"url": "https://..."}`. Returns `extraction_id` and initial `status`. |
| `GET` | `/api/v1/recipe-extractions/{id}` | Poll extraction status. Status values: `queued`, `extracting`, `done`, `failed`. |
| `GET` | `/api/v1/recipes` | List all saved recipes (id + title). |
| `GET` | `/api/v1/recipes/{id}` | Get full recipe details. |
| `DELETE` | `/api/v1/recipes/{id}` | Delete a saved recipe. Returns `204` on success and `404` if the recipe does not exist. |
| `GET` | `/healthz` | Health check. |

## Example: Submit a URL

```bash
curl -X POST http://localhost:8080/api/v1/recipes \
  -H "Content-Type: application/json" \
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
curl http://localhost:8080/api/v1/recipe-extractions/a1b2c3d4-...
```

```json
{
  "id": "a1b2c3d4-...",
  "source_url": "https://example.com/chocolate-chip-cookies",
  "status": "done",
  "recipe_id": "e5f6g7h8-..."
}
```
