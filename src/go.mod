module github.com/honeycombio/opentelemetry-workshop

go 1.13

require (
	github.com/honeycombio/opentelemetry-exporter-go v0.4.0
  github.com/lightstep/opentelemetry-exporter-go v0.1.5
	go.opentelemetry.io/otel v0.4.2
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.4.2
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.4.2
	github.com/GoogleCloudPlatform/opentelemetry-operations-go v0.1.0
	google.golang.org/genproto v0.0.0-20200305110556-506484158171 // indirect
	google.golang.org/grpc v1.27.1
)
