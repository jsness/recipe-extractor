# Utility Scripts

| Script | Description |
|---|---|
| `.\scripts\dev.ps1` | Start isolated Postgres, rebuild/restart the Go server, and start or reuse Vite |
| `.\scripts\clear-db.ps1` | Wipe all data from local Postgres (prompts for confirmation) |

Run `.\scripts\dev.ps1` again after changing Go code. It builds before stopping
the old backend, replaces the recipe-extractor process on the configured dev
port (8081 by default), and reuses this checkout's running Vite server on 5174.
It refuses to stop unrelated processes and reserves port 8080 for deployment.
The backend runs in the background; the script prints its PID and log directory
under your temporary folder. Startup runs pending database migrations.
If Go is absent from the current terminal's PATH, the script also checks its
standard Windows installation locations.
