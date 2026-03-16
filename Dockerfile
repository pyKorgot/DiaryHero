FROM golang:1.25.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/diaryhero ./cmd/diaryhero

FROM alpine:3.22

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app && mkdir -p /app/data && chown -R app:app /app

COPY --from=builder /out/diaryhero /app/diaryhero

USER app

ENV APP_ENV=production
ENV DATABASE_PATH=/app/data/diaryhero.db

VOLUME ["/app/data"]

CMD ["/app/diaryhero"]
