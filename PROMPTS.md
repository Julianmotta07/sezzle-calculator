# Prompts

A log of the AI prompts used while building this project.

## 1. Repository structure (Phase 1)

```
I'm building a full-stack calculator for a Sezzle internship technical assessment: a React + TypeScript frontend (Vite) and a Go REST backend. Create only the initial repository structure: backend/, frontend/ and docs/ folders (each with a .gitkeep), a root .gitignore suitable for Go and Node, a README.md with a title and these empty section headings: Overview, Tech stack, Project structure, Getting started, API, Testing, Design decisions, AI usage. Also create an empty PROMPTS.md with a short intro line. Don't write application code yet.
```

## 2. Calculator domain logic (Phase 2)

```
In backend/, initialize a Go module (github.com/Julianmotta07/sezzle-calculator/backend) and create package internal/calculator with pure functions for: Add, Subtract, Multiply, Divide, Power, Sqrt and Percentage (Percentage(a, b) = a percent of b = a*b/100). Return sentinel errors for invalid cases: division by zero, square root of a negative number, and non-finite results (NaN or ±Inf). Use only the standard library. Write table-driven unit tests in calculator_test.go covering normal cases, edge cases (negatives, decimals, zero) and every error, using errors.Is. Keep functions small and documented.
```

## 3. Code review (used after each step)

```
Before I commit, explain the changes you just made, file by file: what each part does, why you chose this approach, and one alternative you considered. Keep it concise. Then list the commands I should run to verify it works.
```

## 4. HTTP API layer (Phase 2)

```
Now add the HTTP layer in backend/internal/api using only net/http (Go 1.22+ method routing). Endpoints:
- GET /health -> {"status":"ok"}
- GET /api/v1/operations -> list of supported operations and how many operands each needs
- POST /api/v1/calculate with body {"operation":"divide","a":10,"b":2} -> 200 {"operation":"divide","result":5}
Validation: return 400 for malformed JSON, unknown fields, unknown operations and missing operands; return 422 for math errors (division by zero, sqrt of a negative, non-finite result). All errors share one JSON shape: {"error":{"code":"DIVISION_BY_ZERO","message":"..."}}. Limit the request body size. Add a small CORS middleware configured by an ALLOWED_ORIGIN env var (default http://localhost:5173). Create cmd/server/main.go that reads PORT (default 8080) and starts the server with graceful shutdown. Write handler tests with net/http/httptest for the success path and every error path.
```