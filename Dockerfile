# syntax=docker/dockerfile:1

# ── Frontend build ──────────────────────────────────────────────────────────
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ── Backend build (embeds the frontend bundle) ─────────────────────────────
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist/ webdist/dist/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/clustermind ./cmd/clustermind

# ── Runtime (distroless, non-root) ─────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/clustermind /clustermind
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/clustermind"]
CMD ["serve", "--port", "8080"]
