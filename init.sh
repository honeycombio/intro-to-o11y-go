#!/bin/bash

set -e

export GOPATH=$PWD
if [ ! -d /tmp/go ]; then
  cd /tmp
  if [ ! -f /tmp/go1.16.1.linux-amd64.tar.gz ]; then
    wget -q https://golang.org/dl/go1.16.1.linux-amd64.tar.gz
  fi
  sha256sum -c ~/go1.16.1.linux-amd64.tar.gz.SHA256SUMS || (echo "failed to verify go tarball" && rm /tmp/go1.16.1.linux-amd64.tar.gz && exit 1)
  tar -xzf go1.16.1.linux-amd64.tar.gz
  rm /tmp/go1.16.1.linux-amd64.tar.gz
fi

mkdir -p /tmp/pkg
if [ ! -L pkg ]; then
  ln -s /tmp/pkg ~/pkg
fi
cd ~/src

/tmp/go/bin/go run main.go
