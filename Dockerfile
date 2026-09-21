# Stage 1: build the binary with the full Go toolchain
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server .

# Stage 2: minimal runtime image — no compiler, no source, just the binary
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
COPY internal/db/schema.sql ./internal/db/schema.sql
EXPOSE 8080
CMD ["./server"]