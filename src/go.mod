module github.com/honeycombio/opentelemetry-workshop

go 1.13

require (
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.2.0
	github.com/honeycombio/opentelemetry-exporter-go v0.8.0
	github.com/lightstep/opentelemetry-exporter-go v0.6.3
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	go.opentelemetry.io/otel v0.8.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.8.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.8.0
	go.opentelemetry.io/otel/exporters/trace/stackdriver v0.2.3 // indirect
	google.golang.org/grpc v1.30.0
)
