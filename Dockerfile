# Multi-stage distroless build for Digitera Helpdesk

# Stage 1: Build Binary
FROM golang:1.23-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o /helpdesk ./cmd/server

# Stage 2: Distroless Static Container
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /helpdesk /app/helpdesk
COPY --from=builder /build/migrations /app/migrations

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/helpdesk"]
CMD ["serve"]
