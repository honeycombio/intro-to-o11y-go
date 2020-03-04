#!/bin/bash

cd src
GO111MODULE=on GOPATH=~ /tmp/go/bin/go get -u github.com/honeycombio/opentelemetry-exporter-go/honeycomb
GO111MODULE=on GOPATH=~ /tmp/go/bin/go get -u go.opentelemetry.io/otel/exporters/trace/jaeger
GO111MODULE=on GOPATH=~ /tmp/go/bin/go get -u go.opentelemetry.io/otel/exporters/trace/stackdriver
GO111MODULE=on GOPATH=~ /tmp/go/bin/go get -u go.opentelemetry.io/otel
GO111MODULE=on GOPATH=~ /tmp/go/bin/go mod tidy
