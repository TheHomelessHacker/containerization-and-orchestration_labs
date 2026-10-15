FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -o api .

FROM scratch
COPY --from=builder /build/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]