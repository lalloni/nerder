FROM ubuntu:26.04

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential pkg-config git ca-certificates \
        libgtk-3-dev libwebkit2gtk-4.1-dev \
        golang-go nodejs npm \
        xvfb xauth dbus-x11 \
    && rm -rf /var/lib/apt/lists/*

RUN GOBIN=/usr/local/bin GOPATH=/tmp/gopath go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0 \
    && rm -rf /tmp/gopath

WORKDIR /src
