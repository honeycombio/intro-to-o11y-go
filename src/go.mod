module github.com/honeycombio/opentelemetry-workshop

go 1.13

require (
	cloud.google.com/go v0.61.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.2.0
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/google/go-cmp v0.5.1 // indirect
	github.com/honeycombio/opentelemetry-exporter-go v0.9.0
	github.com/lightstep/opentelemetry-exporter-go v0.6.3
	go.opentelemetry.io/otel v0.9.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.8.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.9.0
	golang.org/x/sys v0.0.0-20200728102440-3e129f6d46b1 // indirect
	google.golang.org/genproto v0.0.0-20200728010541-3dc8dca74b7b // indirect
	google.golang.org/grpc v1.30.0
)
