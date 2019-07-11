#!/bin/bash

export GOPATH=$PWD
export GO111MODULE=on
if [ ! -d go ]; then
  cd /tmp
  wget -q https://dl.google.com/go/go1.12.7.linux-amd64.tar.gz
  tar -xzf go1.12.7.linux-amd64.tar.gz
  mv go ~
  rm /tmp/go1.12.7.linux-amd64.tar.gz
fi
cd ~/src
OPENTELEMETRY_LIB=~/stderr.so ~/go/bin/go run main.go
