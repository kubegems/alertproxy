# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY ./ .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o alertproxy main.go

FROM alpine:3.16
WORKDIR /
RUN apk --update add tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata && \
    rm -rf /var/cache/apk/*
COPY --from=builder /workspace/alertproxy .

ENTRYPOINT ["/alertproxy"]
