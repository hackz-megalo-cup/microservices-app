FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY game-server/go.mod game-server/go.sum ./
RUN go mod download
COPY game-server/ ./
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /game-server .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /game-server /game-server
ENTRYPOINT ["/game-server"]
