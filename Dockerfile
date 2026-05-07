# Build
FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /rose-academy .

# Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -g 65532 -S app \
	&& adduser -u 65532 -S -G app -h /app app

WORKDIR /app

COPY --from=builder /rose-academy /app/rose-academy

USER app:app

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/rose-academy"]
