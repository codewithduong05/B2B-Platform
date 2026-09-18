FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /atlas-api ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /atlas-api /atlas-api
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/.env.example .env.example

EXPOSE 8000

ENTRYPOINT ["/atlas-api"]