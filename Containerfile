FROM ubuntu:26.04

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential pkg-config git ca-certificates \
        libgtk-3-dev libwebkit2gtk-4.1-dev \
        curl nodejs npm \
        xvfb xauth dbus-x11 \
    && rm -rf /var/lib/apt/lists/*

# Ubuntu's golang-go lags behind, so install the official toolchain (checksum-verified).
ARG GO_VERSION=1.27.3
ENV GOTOOLCHAIN=local
ENV PATH=/usr/local/go/bin:$PATH
RUN arch="$(dpkg --print-architecture)" \
    && file="go${GO_VERSION}.linux-${arch}.tar.gz" \
    && curl -fsSLO "https://dl.google.com/go/${file}" \
    && echo "$(curl -fsSL "https://dl.google.com/go/${file}.sha256")  ${file}" | sha256sum -c - \
    && tar -C /usr/local -xzf "${file}" \
    && rm "${file}"

RUN GOBIN=/usr/local/bin GOPATH=/tmp/gopath go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0 \
    && rm -rf /tmp/gopath

WORKDIR /src
