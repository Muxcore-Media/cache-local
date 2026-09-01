FROM golang:1.27-alpine AS builder
COPY core/ /build/core/
COPY cache-local/ /build/cache-local/
WORKDIR /build/cache-local
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /module ./cmd/module
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /module /
ENTRYPOINT ["/module"]
