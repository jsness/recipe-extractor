# Stage 1: Build React frontend
FROM node:20-alpine AS frontend
WORKDIR /app
COPY web/package*.json ./web/
RUN cd web && npm ci
COPY web/ ./web/
RUN cd web && npm run build

# Stage 2: Build Go binary with embedded frontend
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
# Copy built frontend so //go:embed can include it
COPY --from=frontend /app/web/dist ./internal/frontend/dist
RUN go build -o /recipe-extractor ./cmd/server

# Stage 3: Minimal runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /recipe-extractor /recipe-extractor
ENTRYPOINT ["/recipe-extractor"]
