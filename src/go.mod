module github.com/honeycombio/opentelemetry-workshop

go 1.13

require (
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.1.0
	github.com/honeycombio/opentelemetry-exporter-go v0.4.3
	github.com/lightstep/opentelemetry-exporter-go v0.1.5
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	go.opentelemetry.io/otel v0.4.3
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.4.3
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.4.3
	go.opentelemetry.io/otel/exporters/trace/stackdriver v0.2.3 // indirect
	golang.org/x/sys v0.0.0-20200501145240-bc7a7d42d5c3 // indirect
	google.golang.org/api v0.23.0 // indirect
	google.golang.org/genproto v0.0.0-20200430143042-b979b6f78d84 // indirect
	google.golang.org/grpc v1.29.1
)
