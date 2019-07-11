package main

import (
  "net/http"
  "fmt"
  "log"

  "github.com/open-telemetry/opentelemetry-go/api/core"
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
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
  attrs, tags, spanCtx := httptrace.Extract(req)

  req = req.WithContext(tag.WithMap(req.Context(), tag.NewMap(core.KeyValue{}, tags, core.Mutator{}, nil)))

  ctx, span := tracer.Start(
    req.Context(),
    "root",
    apitrace.WithAttributes(attrs...),
    apitrace.ChildOf(spanCtx),
  )
  defer span.Finish()
  span.AddEvent(ctx, event.WithString("handling this..."))

  fmt.Fprintf(w, "Ohai world!")
}


func main() {
  http.HandleFunc("/", rootHandler)

  err := http.ListenAndServe(":3000", nil)
  if err != nil {
    log.Fatalf("Could not start web server: %s", err)
  }
}
