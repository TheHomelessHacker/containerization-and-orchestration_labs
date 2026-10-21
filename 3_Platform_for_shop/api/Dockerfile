FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -o app .

FROM scratch
COPY --from=builder /build/app /app
ENTRYPOINT ["/app"]