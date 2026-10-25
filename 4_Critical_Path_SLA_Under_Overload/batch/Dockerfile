FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY main.go ./
RUN CGO_ENABLED=0 go build -o batch main.go

FROM scratch
COPY --from=builder /build/batch /batch
ENTRYPOINT ["/batch"]