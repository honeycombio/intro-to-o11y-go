#!/bin/bash

cd src
GO111MODULE=on GOPATH=~ /tmp/go/bin/go get -u go.opentelemetry.io/otel
GO111MODULE=on GOPATH=~ /tmp/go/bin/go mod tidy
