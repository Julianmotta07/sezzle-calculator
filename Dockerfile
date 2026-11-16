# syntax=docker/dockerfile:1

# Full-stack image: the Go server serves both the REST API and the built React app.
#   docker build -t sezzle-calculator .
#   docker run --rm -p 8080:8080 sezzle-calculator   ->  http://localhost:8080

# ---- Stage 1: build the React frontend ----------------------------------------
FROM node:24-alpine AS frontend
WORKDIR /src/frontend

# Install dependencies first so this layer is cached until the lockfile changes.
# npm ci installs exactly what package-lock.json lists (reproducible) and fails
# if the lockfile is incomplete for Linux, e.g. after a Windows `npm install`
# drops platform-specific optional deps. See README > Troubleshooting.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci || { echo >&2 "npm ci failed: frontend/package-lock.json is out of sync for Linux. See README > Troubleshooting."; exit 1; }

COPY frontend/ ./
# Empty base URL: the browser calls /api/v1/... on the same origin that served the page.
ENV VITE_API_URL=""
RUN npm run build

# ---- Stage 2: build a static Go binary -----------------------------------------
FROM golang:1.25-alpine AS backend
WORKDIR /src/backend

# go.sum is optional (no third-party modules yet); the glob keeps COPY from failing.
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

COPY backend/ ./
# CGO disabled -> fully static binary that runs on a distroless/scratch base.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- Stage 3: minimal runtime image ----------------------------------------------
# distroless/static: no shell or package manager, CA certs and tzdata included,
# runs as an unprivileged "nonroot" user by default.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=backend /out/server /app/server
COPY --from=frontend /src/frontend/dist /app/static

ENV PORT=8080 \
    STATIC_DIR=/app/static
EXPOSE 8080

ENTRYPOINT ["/app/server"]
