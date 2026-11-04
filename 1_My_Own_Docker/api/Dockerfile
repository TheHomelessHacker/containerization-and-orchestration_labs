FROM golang:1.22-alpine
WORKDIR /build
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -o api .
EXPOSE 8080
ENTRYPOINT ["/build/api"]
