---
name: cloud-agent-runbook
description: Practical setup, run, and test instructions for Cursor Cloud agents working on this Go and Next.js Japan Post zipcode proxy monorepo.
---

# Cloud Agent Runbook

Use this skill when you need to run, test, or manually verify this codebase in Cursor Cloud.

## Repository layout

- `backend/`: Go HTTP API. Main endpoint is `POST /api/v1/search/zipcode`; health check is `GET /healthz`.
- `frontend/`: Next.js UI that calls the backend through `NEXT_PUBLIC_API_BASE_URL`.
- `docker-compose.yml`: Starts both services with root `.env`.
- `.env.example`: Copy this to `.env` before Docker or local app runs.

## Credentials, login, and flags

- There is no product login flow in this app.
- Real Japan Post API calls require `JAPANPOST_CLIENT_ID` and `JAPANPOST_SECRET_KEY` in `.env` or exported env vars. Never commit real values.
- For most Cloud work, prefer mocked tests or a local stub API instead of real Japan Post credentials.
- There are no feature flags in the frontend. Runtime behavior is controlled by env vars:
  - `JAPANPOST_API_BASE_URL`: point backend at production, test, or a local stub.
  - `JAPANPOST_TOKEN_PATH`: default `/api/v2/j/token`; set to `/api/v1/j/token` for older upstreams.
  - `JAPANPOST_SEARCH_CODE_PATH`: default `/api/v2/searchcode`; set to `/api/v1/searchcode` for older upstreams.
  - `JAPANPOST_TOKEN_SCOPE`: set to values such as `J1` only when the target environment requires it.
  - `JAPANPOST_X_FORWARDED_FOR`: default `127.0.0.1`; required by the upstream API.
  - `CORS_ALLOW_ORIGIN`: default `*`; set to the frontend origin when testing stricter CORS.
  - `NEXT_PUBLIC_API_BASE_URL`: frontend-visible backend URL; set it to the backend URL used in the current run.

## Whole app with Docker Compose

1. Create the env file:
   - `cp .env.example .env`
2. Fill real Japan Post credentials only when the task explicitly needs live upstream testing.
3. Start both services:
   - `docker compose up --build`
4. Verify:
   - Backend health: `curl -i <backend-url>/healthz`
   - Frontend: open `http://localhost:3000`

If live credentials are unavailable, Docker can still validate service startup and health, but zipcode search will fail when it reaches the upstream API.

## Backend workflow

Run from `backend/`.

- Build check: `go build ./...`
- Static analysis: `go vet ./...`
- Unit and integration tests with mocked upstreams: `go test ./...`
- Focused handler tests: `go test ./internal/handler/...`
- Local server with real or stub credentials:
  - `export JAPANPOST_CLIENT_ID=test-client`
  - `export JAPANPOST_SECRET_KEY=test-secret`
  - `export JAPANPOST_API_BASE_URL=http://127.0.0.1:<stub-port>`
  - `export JAPANPOST_X_FORWARDED_FOR=127.0.0.1`
  - `go run ./cmd/server`

Concrete backend smoke test once the server is running:

- `curl -i <backend-url>/healthz`
- `curl -i -X POST <backend-url>/api/v1/search/zipcode -H 'Content-Type: application/json' -d '{"zipcode":"100-0001","page":1,"limit":10,"choikitype":1,"searchtype":1}'`

When changing Japan Post client behavior, add or update `httptest.Server` based tests under `backend/internal/client/`. Existing tests already mock token and searchcode responses.

## Frontend workflow

Run from `frontend/`.

- Install dependencies: `npm install`
- Development server: set `NEXT_PUBLIC_API_BASE_URL` to the backend URL, then run `npm run dev`.
- Production build: `npm run build`
- Lint caveat: `npm run lint` currently calls `next lint`; with Next.js 16 this can fail with `Invalid project directory provided, no such directory: .../lint`. Prefer `npm run build` for TypeScript/build validation until the lint script is updated.

Concrete UI testing workflow:

1. Start the backend first, using either real credentials or a local stub.
2. Set `NEXT_PUBLIC_API_BASE_URL` to the backend URL and run `npm run dev`.
3. Open `http://localhost:3000`.
4. Submit `100-0001` and verify the UI shows either formatted JSON from the backend or a clear backend/upstream error.
5. For UI changes, use browser-based manual testing and save a short walkthrough video.

## Local upstream stub pattern

Use a stub when you need end-to-end backend/frontend behavior without real Japan Post credentials.

- The stub must implement:
  - `POST /api/v2/j/token` returning `{"token":"stub-token","expires_in":3600,"token_type":"Bearer"}`
  - `GET /api/v2/searchcode/{zipcode}` returning representative JSON
- Point `JAPANPOST_API_BASE_URL` at the stub and keep the default v2 paths unless the task is specifically about v1 compatibility.
- Keep ad hoc stubs outside committed code unless the task asks for reusable test tooling.

## Updating this skill

Update this skill whenever you discover a repeatable runbook detail that saves future agents time:

- Add exact commands that worked in Cursor Cloud.
- Note environment variables, stubs, ports, or credentials assumptions that were required.
- Record known failures and fixes, especially dependency, Docker, Go toolchain, Next.js, or upstream API quirks.
- Keep the skill short and actionable; link to larger docs instead of copying long explanations.
