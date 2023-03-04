FROM golang:1.20.1-alpine3.17 AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /src

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed
RUN go mod download

# Copy source code
COPY . .

# Build the Go app
RUN go build -o /go/bin/app

FROM alpine:3.17

COPY --from=builder /go/bin/app /go/bin/app

CMD ["/go/bin/app"]