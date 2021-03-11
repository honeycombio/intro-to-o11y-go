module github.com/honeycombio/opentelemetry-workshop

go 1.13

require (
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/klauspost/compress v1.11.1 // indirect
	github.com/prometheus/client_golang v1.8.0 // indirect
	github.com/vmihailenco/tagparser v0.1.2 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.13.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.13.0
	go.opentelemetry.io/otel v0.18.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.13.0
	go.opentelemetry.io/otel/exporters/stdout v0.13.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.18.0
	go.opentelemetry.io/otel/sdk v0.18.0
	google.golang.org/api v0.41.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
