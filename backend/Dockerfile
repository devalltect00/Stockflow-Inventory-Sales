# Build a single Go executable, then run it as an unprivileged user.
FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /stockflow ./cmd/api

FROM alpine:3.23
RUN addgroup -S app && adduser -S app -G app
COPY --from=build /stockflow /usr/local/bin/stockflow
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s CMD wget -q -O /dev/null http://127.0.0.1:8080/api/health || exit 1
CMD ["stockflow"]
