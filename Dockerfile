FROM golang:1.23.0-alpine3.20 AS builder
MAINTAINER "rensongqi1024@gmail.com"
USER root
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn
WORKDIR /opt
COPY . .
RUN go build -ldflags="-s -w" -o gpfs-fuse .

FROM docker.io/ubuntu:22.04-fuse
USER root
COPY --from=builder /opt/gpfs-fuse /usr/bin/gpfs-fuse
ENV GPFS_FILESYSTEM=upload
ENTRYPOINT ["/usr/bin/gpfs-fuse", "-external", "/upload", "-mount", "/disk/upload"]