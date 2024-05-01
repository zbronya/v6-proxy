#!/bin/bash

latest_release=$(curl --silent "https://api.github.com/repos/zbronya/v6-proxy/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

supported_os=("linux")
supported_arch=("amd64" "arm64")

os=$(uname -s | tr '[:upper:]' '[:lower:]')
if [[ ! " ${supported_os[*]} " =~ " ${os} " ]]; then
    echo "Unsupported OS: ${os}"
    exit 1
fi

arch=$(uname -m)
case "$arch" in
    x86_64)
        arch="amd64"
        ;;
    aarch64 | armv8*)
        arch="arm64"
        ;;
    *)
        echo "Unsupported architecture: ${arch}"
        exit 1
        ;;
esac

if [[ ! " ${supported_arch[*]} " =~ " ${arch} " ]]; then
    echo "Unsupported architecture: ${arch}"
    exit 1
fi

curl -sOL "https://github.com/zbronya/v6-proxy/releases/download/${latest_release}/v6-proxy-${os}-${arch}"

chmod +x "v6-proxy-${os}-${arch}"

sudo mv "v6-proxy-${os}-${arch}" /usr/local/bin/v6-proxy
