# URL Shortener

A minimal URL shortener written in Go. Paste a long URL, get a short one. No database, no dependencies — just the standard library and an in-memory store.

## Endpoints

| Method | Path       | Description                        |
|--------|------------|------------------------------------|
| GET    | `/`        | Web UI                             |
| POST   | `/shorten` | Shorten a URL (JSON API)           |
| GET    | `/{code}`  | Redirect to the original URL       |

## Using the UI

1. Start the server (see [Development](#development) below)
2. Open `http://localhost:8080` in your browser
3. Paste any URL into the input field and click **Shorten** (or press Enter)
4. Copy the short URL from the result and share it — visiting it will redirect to the original

## Development

```bash
go run main.go
```

The server starts on `http://localhost:8080`. Open that in a browser to use the UI, or hit the API directly:

```bash
# Shorten a URL
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# Follow a short link
curl -L http://localhost:8080/<code>
```

## Testing

```bash
go test ./...
```

## Building for Production

```bash
go build -o url-shortener .
./url-shortener
```

Or as a single command:

```bash
CGO_ENABLED=0 GOOS=linux go build -o url-shortener . && ./url-shortener
```

The `CGO_ENABLED=0` flag produces a fully static binary suitable for scratch/distroless containers.

### Docker

```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o url-shortener .

FROM scratch
COPY --from=builder /app/url-shortener /url-shortener
EXPOSE 8080
ENTRYPOINT ["/url-shortener"]
```

```bash
docker build -t url-shortener .
docker run -p 8080:8080 url-shortener
```

> **Note:** The in-memory store is not persisted. All shortened URLs are lost on restart.
