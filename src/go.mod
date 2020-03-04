module github.com/honeycombio/opentelemetry-workshop

go 1.13

require (
	github.com/honeycombio/libhoney-go v1.12.2 // indirect
	github.com/honeycombio/opentelemetry-exporter-go v0.2.1
	github.com/klauspost/compress v1.10.2 // indirect
	go.opentelemetry.io/otel v0.2.3
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.2.3
	go.opentelemetry.io/otel/exporters/trace/stackdriver v0.2.3
	google.golang.org/genproto v0.0.0-20200304201815-d429ff31ee6c // indirect
	google.golang.org/grpc v1.27.1
)
