#!/bin/bash

cd src
GO111MODULE=on GOPATH=~ /tmp/go/bin/go get -u github.com/honeycombio/opentelemetry-exporter-go/honeycomb
GO111MODULE=on GOPATH=~ /tmp/go/bin/go get -u go.opentelemetry.io/exporter/trace/jaeger
GO111MODULE=on GOPATH=~ /tmp/go/bin/go get -u go.opentelemetry.io
