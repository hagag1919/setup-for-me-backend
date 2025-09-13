# SetupForMe Backend (Go)

A minimal Go backend for SetupForMe that provides authentication (JWT), app management, script generation, and integration with winget.run. Uses PostgreSQL.

## Tech Stack
- Go (net/http, database/sql)
- PostgreSQL (`lib/pq`)
- JWT (`github.com/golang-jwt/jwt/v5`)
- Dotenv (`github.com/joho/godotenv`)

## Features
- Signup/Login with hashed passwords (bcrypt)
- JWT-protected app CRUD endpoints
- Generate PowerShell install script for a user’s apps
- winget.run integration to auto-resolve Winget IDs
- CORS enabled for local dev

## Prerequisites
- Go 1.20+
- PostgreSQL 13+ (local or container)

Optional (Docker):
```
docker run --name setupforme-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=setupforme -p 5432:5432 -d postgres:15
```

## Configuration
Create `backend/.env` (or set system env vars):
```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/setupforme?sslmode=disable
JWT_SECRET=change-me-in-prod
PORT=8080
```
Notes:
- `DATABASE_URL` must match your DB credentials.
- Tables are auto-created on startup (users, apps).

## Run
From the `backend/` folder:
```
go run .
```
Or build:
```
go build -o setupforme.exe
./setupforme.exe
```
You should see: `Server starting on port 8080`.

## API
Base URL: `/api`

Auth:
- `POST /api/auth/signup` { email, password }
- `POST /api/auth/login`  { email, password }

Apps (JWT required – `Authorization: Bearer <token>`):
- `GET    /api/apps` – list apps for current user
- `POST   /api/apps` – create `{ name, winget_id?, download_url?, args? }`
  - If `winget_id` and `download_url` are missing, server will try to resolve `winget_id` from winget.run using `name`.
- `PUT    /api/apps/{id}` – update
- `DELETE /api/apps/{id}` – delete
- `GET    /api/apps/script` – returns `{ message, data: { script } }`

Winget search:
- `GET /api/winget/search?q=<query>` – returns top match (id/name) for suggestions

## Database
Tables are created on startup:
- `users (id SERIAL PK, email UNIQUE, password)`
- `apps  (id SERIAL PK, user_id FK, name, winget_id, download_url, args)`

## Script Generation
- Generates a PowerShell script per user apps
- Prefers `winget install -e --id <ID> --accept-*`
- Falls back to downloading and executing URL if provided
- Per-app try/catch to avoid aborting the whole run

## CORS
CORS allows localhost dev origins (`5173`, `3000`) and sets headers for `Content-Type, Authorization`. OPTIONS preflight returns 200.

## Troubleshooting
- 409 on signup: user already exists – login instead or delete from DB.
- 500 on apps endpoints: ensure PostgreSQL is running and `DATABASE_URL` is correct; verify SQL placeholders are `$1..$n` (PostgreSQL).
- 401 on protected routes: ensure `Authorization: Bearer <token>` is included.
- winget.run lookup failures: either provide `winget_id` manually or set a direct `download_url`.

## License
MIT (see repository root if present)
