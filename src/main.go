package main

import (
  "context"
  "net/http"
  "fmt"
  "log"
  "os"

  "github.com/open-telemetry/opentelemetry-go/api/core"
 	"github.com/open-telemetry/opentelemetry-go/api/metric"
	"github.com/open-telemetry/opentelemetry-go/api/stats"
	"github.com/open-telemetry/opentelemetry-go/api/tag"
	apitrace "github.com/open-telemetry/opentelemetry-go/api/trace"
	"github.com/open-telemetry/opentelemetry-go/plugin/httptrace"

	_ "github.com/open-telemetry/opentelemetry-go/exporter/loader"
	"github.com/open-telemetry/opentelemetry-go/sdk/event"
	"github.com/open-telemetry/opentelemetry-go/sdk/trace"
)

var (
	tracer = trace.Register().
		WithService("server").
		WithComponent("main").
		WithResources(
			tag.New("whatevs").String("nooooo"),
		)
  appKey = tag.New("honeycomb.io/glitch/app", tag.WithDescription("The Glitch app name."))
  containerKey = tag.New("honeycomb.io/glitch/container_id", tag.WithDescription("The Glitch container id."))
	diskMetric = metric.NewFloat64Gauge("honeycomb.io/glitch/disk_usage",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of disk used."),
	)
  diskMetric = metric.NewFloat64Gauge("honeycomb.io/glitch/disk_quota",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of disk quota available."),
	)
)

func dbHandler(ctx context.Context, color string) int {
  ctx, span := tracer.Start(ctx, "database")
  defer span.Finish()

  // Pretend we talked to a database here.
  return 0
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
  attrs, tags, spanCtx := httptrace.Extract(req)

  req = req.WithContext(tag.WithMap(req.Context(), tag.NewMap(core.KeyValue{}, tags, core.Mutator{}, nil)))

  ctx, span := tracer.Start(
    req.Context(),
    "root",
    apitrace.WithAttributes(attrs...),
    apitrace.ChildOf(spanCtx),
  )
  defer span.Finish()

  span.AddEvent(ctx, event.WithString("annotation within span"))
  _ = dbHandler(ctx, "foo")

  fmt.Fprintf(w, "Click [Tools] > [Logs] to see spans!")
}

func fibHandler(w http.ResponseWriter, req *http.Request) {
  // call ourselves minus one for some recursion and complex spans.
}

func main() {
  http.HandleFunc("/", rootHandler)
  os.Stderr.WriteString("Initializing the server...\n")

  
  err := http.ListenAndServe(":3000", nil)
  if err != nil {
    log.Fatalf("Could not start web server: %s", err)
  }
}
