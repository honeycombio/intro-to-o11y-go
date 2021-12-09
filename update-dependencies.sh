#!/bin/bash

cd src
GO111MODULE=on GOPATH=~ go get -u go.opentelemetry.io/otel
GO111MODULE=on GOPATH=~ go mod tidy
