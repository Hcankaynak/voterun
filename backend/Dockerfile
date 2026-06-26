# ---- Build stage ----
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Build a fully static binary. The SQLite driver (modernc.org/sqlite) is
# pure Go, so CGO is not required.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /voterun .

# ---- Runtime stage ----
FROM alpine:3.20

# ca-certificates for any outbound TLS; wget is used by the healthcheck.
RUN apk add --no-cache ca-certificates wget

WORKDIR /app
COPY --from=build /voterun /app/voterun

# Persist the SQLite database in a dedicated directory that can be backed
# by a Docker volume so data survives container/redeploys.
ENV DB_PATH=/data/voterun.db
RUN mkdir -p /data
VOLUME ["/data"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/voterun"]
