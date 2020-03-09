module github.com/honeycombio/opentelemetry-workshop

go 1.13

require (
	github.com/honeycombio/opentelemetry-exporter-go v0.2.3
	go.opentelemetry.io/otel v0.2.3
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.2.3
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.2.3
	go.opentelemetry.io/otel/exporters/trace/stackdriver v0.2.3
	google.golang.org/genproto v0.0.0-20200305110556-506484158171 // indirect
	google.golang.org/grpc v1.27.1
)
