FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/incidenthub ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home incidenthub
WORKDIR /app
COPY --from=build /out/incidenthub /app/incidenthub
COPY templates /app/templates
COPY static /app/static
RUN mkdir -p /app/data && chown -R incidenthub:incidenthub /app
USER incidenthub
ENV ADDR=:8080 DATABASE_PATH=/app/data/incidenthub.db
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/incidenthub"]
