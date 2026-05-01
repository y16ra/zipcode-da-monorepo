# AGENTS.md

## Cursor Cloud specific instructions

### Overview

This is a monorepo with a Go backend (stdlib only, no external deps) and a Next.js 16 frontend that proxies the Japan Post zipcode/digital address API. No database or additional services are required.

### Prerequisites

- **Go 1.26+** (`backend/go.mod` specifies `go 1.26`)
- **Node.js 22+** (frontend uses Next.js 16 / React 19)
- Environment variables `JAPANPOST_CLIENT_ID` and `JAPANPOST_SECRET_KEY` must be set for the backend to start (hard validation in `config.Load()`).

### Running services

**Backend** (port 8080):
```bash
cd backend && go run ./cmd/server
```
Health check: `curl localhost:8080/healthz`

**Frontend** (port 3000):
```bash
cd frontend && npm run dev
```
Set `NEXT_PUBLIC_API_BASE_URL` to point at the backend (port 8080 on localhost by default) if not already in environment.

### Testing

- Backend: `cd backend && go test ./...` (all tests use mocks, no external API needed)
- Backend lint: `cd backend && go vet ./...`
- Frontend build: `cd frontend && npm run build`
- Frontend lint: `npm run lint` is configured to run `next lint`, but **Next.js 16 removed the built-in `next lint` command**. The eslint config (`eslint.config.mjs`) has a circular reference issue with the current `eslint-config-next` version. This is a pre-existing issue in the repo.

### Gotchas

- The backend will **refuse to start** without `JAPANPOST_CLIENT_ID` and `JAPANPOST_SECRET_KEY` env vars. For testing purposes, Go unit tests mock the external API so real credentials are not needed for `go test`.
- Frontend uses `output: "standalone"` in `next.config.ts` for production Docker builds but `npm run dev` works normally for development.
- There is no `go.work` file despite CLAUDE.md mentioning one; the backend is a standalone Go module.
