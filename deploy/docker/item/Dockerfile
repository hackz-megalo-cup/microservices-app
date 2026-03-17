FROM golang:1.26-alpine AS builder
WORKDIR /src/services
COPY services/go.mod services/go.sum ./
RUN go mod download
COPY services ./
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /item ./cmd/item

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /item /item
ENTRYPOINT ["/item"]
