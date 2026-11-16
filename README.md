# Sezzle Calculator

A full-stack calculator: a React + TypeScript single-page app that sends calculations to a Go REST API.

## Overview

- **Operations:** add, subtract, multiply, divide, power, square root and percentage.
- **Backend:** a Go service with a single `POST /api/v1/calculate` endpoint, strict input validation and one consistent JSON error format.
- **Frontend:** a responsive, accessible form. It validates input before sending it and turns API errors into friendly messages.
- **Deployment:** one Docker image runs everything. The Go binary serves both the API and the built frontend.

## Tech stack

| Layer    | Technology |
|----------|------------|
| Backend  | Go 1.25, standard library only (`net/http` with Go 1.22+ method routing, `encoding/json`, `log/slog`) |
| Frontend | React 19, TypeScript 7, Vite 8, plain CSS |
| Testing  | Go `testing` + `httptest`; Vitest 5, React Testing Library, jsdom, `@vitest/coverage-v8` |
| Tooling  | oxlint, Docker (multi-stage build, distroless runtime image) |

## Project structure

```
.
├── backend/
│   ├── cmd/server/            # main: env config, HTTP server, graceful shutdown
│   └── internal/
│       ├── calculator/        # pure math functions + sentinel errors
│       └── api/               # routing, handlers, validation, CORS, static file serving
├── frontend/
│   └── src/
│       ├── api/               # typed API client (calculate, ApiError, NetworkError)
│       ├── calculator/        # input parsing, number formatting, operation metadata, error messages
│       ├── hooks/             # useCalculator: form state + API call
│       └── components/        # Calculator, OperationSelector, NumberField, ResultDisplay
├── docs/                      # coverage reports
├── Dockerfile                 # builds frontend + backend into one image
├── PROMPTS.md                 # AI prompts used during development
└── README.md
```

## Prerequisites

- **Go 1.25+**
- **Node.js 22.12+ or 24** (required by Vitest 5), with npm
- **Docker** (optional; only needed to run the single-container build)

## Getting started

### Backend

```sh
cd backend
go run ./cmd/server          # listens on http://localhost:8080
```

| Variable         | Default                 | Purpose |
|------------------|-------------------------|---------|
| `PORT`           | `8080`                  | Port to listen on |
| `ALLOWED_ORIGIN` | `http://localhost:5173` | Browser origin allowed by CORS (`*` allows any origin) |
| `STATIC_DIR`     | *(unset)*               | Folder of built frontend files to serve; when unset, the server runs the API only |

Ctrl+C (SIGINT) or SIGTERM stops the server gracefully: requests already in progress get up to 10 seconds to finish.

### Frontend

In a second terminal:

```sh
cd frontend
npm install
npm run dev                  # http://localhost:5173
```

By default the dev build calls the API at `http://localhost:8080`. To change that, set `VITE_API_URL` in `frontend/.env.local` (see [`.env.example`](frontend/.env.example)). An empty value sends requests to the same origin that served the page.

### Docker (full stack, one container)

```sh
docker build -t sezzle-calculator .
docker run --rm -p 8080:8080 sezzle-calculator
```

Open http://localhost:8080. The image builds the frontend with Node, builds a static Go binary, and runs that binary on `distroless/static` as a non-root user, with `STATIC_DIR=/app/static`.

The image has been built with `docker build --no-cache` and checked end to end:
- `/health`, the UI and `POST /api/v1/calculate` all respond correctly from the container.
- The final image is about **15 MB**.

## API

Base URL: `http://localhost:8080`. Request and response bodies are JSON.

> The examples below use bash syntax. In Windows PowerShell, call `curl.exe` and escape the inner double quotes, or use Git Bash.

### `GET /health`

```sh
curl http://localhost:8080/health
```
```json
{"status":"ok"}
```

### `GET /api/v1/operations`

Lists the supported operations and how many operands each one needs.

```sh
curl http://localhost:8080/api/v1/operations
```
```json
{"operations":[
  {"name":"add","operands":2,"description":"a + b"},
  {"name":"subtract","operands":2,"description":"a - b"},
  {"name":"multiply","operands":2,"description":"a * b"},
  {"name":"divide","operands":2,"description":"a / b"},
  {"name":"power","operands":2,"description":"a raised to the power b"},
  {"name":"sqrt","operands":1,"description":"square root of a"},
  {"name":"percentage","operands":2,"description":"a percent of b (a*b/100)"}
]}
```

### `POST /api/v1/calculate`

Request body: `{"operation": string, "a": number, "b": number}`. Leave out `b` for `sqrt`.

**Success (200)**

```sh
curl -X POST http://localhost:8080/api/v1/calculate \
  -H "Content-Type: application/json" \
  -d '{"operation":"divide","a":10,"b":2}'
```
```json
{"operation":"divide","result":5}
```

```sh
curl -X POST http://localhost:8080/api/v1/calculate -d '{"operation":"sqrt","a":9}'
```
```json
{"operation":"sqrt","result":3}
```

**Math error (422)**

```sh
curl -X POST http://localhost:8080/api/v1/calculate -d '{"operation":"divide","a":1,"b":0}'
```
```json
{"error":{"code":"DIVISION_BY_ZERO","message":"cannot divide by zero"}}
```

**Invalid request (400)**

```sh
curl -X POST http://localhost:8080/api/v1/calculate -d '{"operation":"divide","a":10}'
```
```json
{"error":{"code":"MISSING_OPERAND","message":"operation \"divide\" requires field \"b\""}}
```

```sh
curl -X POST http://localhost:8080/api/v1/calculate -d '{"operation":"modulo","a":1,"b":2}'
```
```json
{"error":{"code":"UNKNOWN_OPERATION","message":"unknown operation \"modulo\"; see GET /api/v1/operations"}}
```

### Error codes

Every error uses the same shape: `{"error":{"code":"...","message":"..."}}`.

| Status | Code | When |
|--------|------|------|
| 400 | `INVALID_JSON` | Empty body, malformed JSON, a body that isn't an object, wrong field types, a number outside the float64 range, or extra data after the object |
| 400 | `UNKNOWN_FIELD` | A field other than `operation`, `a` or `b` |
| 400 | `MISSING_OPERATION` / `UNKNOWN_OPERATION` | `operation` missing, or not in the list (names are case-sensitive) |
| 400 | `MISSING_OPERAND` / `UNEXPECTED_OPERAND` | `a` or `b` missing or `null`; `b` sent with `sqrt` |
| 413 | `PAYLOAD_TOO_LARGE` | Body larger than 4 KiB |
| 422 | `DIVISION_BY_ZERO` | `b` is 0 in `divide` |
| 422 | `NEGATIVE_SQRT` | `sqrt` of a negative number |
| 422 | `NON_FINITE_RESULT` | The result is NaN or ±Inf (overflow, `0^-1`, `(-8)^0.5`) |
| 404 / 405 | `NOT_FOUND` / `METHOD_NOT_ALLOWED` | Unknown path, or wrong HTTP method (405 also sets an `Allow` header) |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

## Testing

### Backend

```sh
cd backend
go vet ./...
go test ./...                                   # 164 test cases, including subtests
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out                # per-function summary
go tool cover -html=coverage.out -o ../docs/backend-coverage.html
```

The tests cover:
- The calculator functions, with normal values, edge cases and every error.
- Every handler success and error path, using `httptest`.
- CORS, static file serving (including path traversal) and routing.
- `run()` itself: it starts on a random port, answers requests, and shuts down when its context is cancelled.

### Frontend

```sh
cd frontend
npm test                     # 61 tests
npm run test:coverage        # text summary + HTML report in frontend/coverage/
```

The tests cover:
- The API client, with `fetch` mocked: success, an API error, a network failure, and how the base URL is chosen.
- Input parsing, number formatting and the friendly error messages.
- The UI, using React Testing Library: a successful calculation, input validation and focus, the loading state, API errors, and hiding the second input for `sqrt`.

### Current coverage ([docs/](docs/))

| Layer | Scope | Coverage |
|-------|-------|----------|
| Backend | `internal/calculator` | 100% |
| Backend | `internal/api` | 100% |
| Backend | `cmd/server` | 75% (`main` itself is untested; `run` is at 88.9%) |
| Backend | **Total (statements)** | **95.3%** |
| Frontend | **All files** | **96.42% statements, 95.87% branches, 95.12% functions, 97.63% lines** |

Full reports: [backend-coverage.txt](docs/backend-coverage.txt), [backend-coverage.html](docs/backend-coverage.html), [frontend-coverage.txt](docs/frontend-coverage.txt).

`main()` only wires OS signals to `run()` and calls `os.Exit`, so testing it would mean starting a separate process and sending it real signals. Everything it delegates to is tested through `run()`.

## Design decisions and assumptions

- **One `/calculate` endpoint instead of one per operation.** The operation is part of the request body, which keeps the API small and lets the client treat every operation the same way. All operations are defined in one list, which drives both validation and `GET /api/v1/operations`, so the two can't drift apart.
- **400 vs 422.**
  - **400** means the request itself is wrong: bad JSON, unknown fields, an unknown operation, or a missing operand.
  - **422** means the request is valid but the math has no finite answer: division by zero, square root of a negative, or an overflow.
  - This lets clients tell a bug in their own code apart from something the user typed.
- **Percentage definition.** `percentage(a, b)` is **a percent of b**, which is `a * b / 100`. For example, `20` and `50` give `10`.
- **Standard library only (backend).** Go 1.22+ routing covers method matching, and `encoding/json` with `DisallowUnknownFields` plus `http.MaxBytesReader` covers strict decoding. The backend has no third-party dependencies, so there's nothing external to audit or update.
- **Pure calculator package.** `internal/calculator` has no HTTP or I/O code. Each function returns a value or a sentinel error (`ErrDivisionByZero`, `ErrNegativeSqrt`, `ErrNonFinite`), and the API layer maps those errors to HTTP responses. The math is tested without HTTP, and the HTTP layer just translates.
- **Strict validation.**
  - Missing operands are detected separately from an explicit `0`.
  - `sqrt` rejects a `b` instead of silently ignoring it.
  - Results that are NaN or ±Inf are rejected, because standard JSON can't represent them.
- **Same-origin serving in Docker.** The production frontend is built with an empty `VITE_API_URL`, so it calls `/api/v1/...` on whatever host served the page. The Go server serves the built files through `STATIC_DIR`. In the container that means no CORS setup and one port. Unknown `/api/...` paths still return JSON 404s, not the HTML page.
- **Assumptions.**
  - Numbers are IEEE-754 `float64` on both ends, which is enough for a general-purpose calculator. The UI rounds what it displays to 15 significant digits, so `0.1 + 0.2` shows as `0.3`.
  - The UI accepts a dot as the decimal separator, not a comma.
  - The frontend validates input as a convenience for the user; the backend remains the source of truth.

## Troubleshooting

**`docker build` fails at `npm ci` with `Missing: @emnapi/core@… from lock file`.**

- **Why it happens:** Vite 8's bundler has a WebAssembly fallback build whose optional dependencies are `@emnapi/core` and `@emnapi/runtime`. When npm rewrites `package-lock.json` on Windows, it can drop those entries (a known npm bug). `npm ci` on Linux then rejects the lockfile.
- **Current fix:** both packages are pinned as exact `devDependencies` in `frontend/package.json`, so npm keeps them in the lockfile on every platform.
- **If it happens again** (for example, after a Vite upgrade needs a different `@emnapi` version):
  1. Set the pinned versions to the ones listed under `node_modules/@rolldown/binding-wasm32-wasi` in the lockfile.
  2. Run `npm install` in `frontend/`.
  3. Check the lockfile on Linux:

```sh
cd frontend
docker run --rm -v "$PWD":/src:ro node:24-alpine sh -c "cp -r /src /app && cd /app && npm ci --ignore-scripts"
```

## Possible improvements

- **Arbitrary-precision arithmetic:** use `math/big` or a decimal type for exact decimal results.
- **More operations:** modulo, n-th root, logarithms, or full expression parsing such as `2 * (3 + 4)`.
- **Calculation history:** store past calculations on the client or the server.
- **Rate limiting and request logging middleware**, with request IDs.
- **Machine-readable API docs:** an OpenAPI spec, plus contract tests that check the frontend types against it.
- **End-to-end tests** with Playwright, run against the Docker image.
- **CI pipeline** with GitHub Actions: lint, test, enforce a coverage threshold, and build the image.
- **Container health check:** a `/health` probe in the orchestrator. The distroless image has no `curl`.

## AI usage

I used an AI coding assistant (Claude Code) throughout the project. It worked phase by phase from specific prompts, and I asked it to explain each change before every commit. I reviewed, ran and verified each step (tests, coverage, manual checks) before committing. The full list of prompts is in [PROMPTS.md](PROMPTS.md).
