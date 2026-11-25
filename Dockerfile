# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS builder

WORKDIR /src/api

RUN apk add --no-cache ca-certificates tzdata

COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api/ ./

ARG TARGETOS=linux
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/ecommerce-api ./cmd/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/ecommerce-api /app/ecommerce-api

ENV API_PORT=:5000 \
    GIN_MODE=release

EXPOSE 5000

USER appuser

ENTRYPOINT ["/app/ecommerce-api"]
