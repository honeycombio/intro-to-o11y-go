package main

import (
  "context"
  "net/http"
  "fmt"
  "log"
  "os"
  "syscall"
  "time"

  "go.opentelemetry.io/api/core"
 	"go.opentelemetry.io/api/metric"
	"go.opentelemetry.io/api/tag"
	apitrace "go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/plugin/httptrace"

	_ "go.opentelemetry.io/exporter/loader"
	"go.opentelemetry.io/sdk/event"
	"go.opentelemetry.io/sdk/trace"
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
	diskUsedMetric = metric.NewFloat64Gauge("honeycomb.io/glitch/disk_usage",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of disk used."),
	)
  diskQuotaMetric = metric.NewFloat64Gauge("honeycomb.io/glitch/disk_quota",
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

func updateDiskMetrics(ctx context.Context, used, quota *metric.Float64Entry) {
  for {
    var stat syscall.Statfs_t
    wd, _ := os.Getwd()
    syscall.Statfs(wd, &stat)

    all := float64(stat.Blocks) * float64(stat.Bsize)
    free := float64(stat.Bfree) * float64(stat.Bsize)
    used.Set(ctx, all - free)
    quota.Set(ctx, all)
    time.Sleep(time.Minute)
  }
}

func main() {
  http.HandleFunc("/", rootHandler)
  os.Stderr.WriteString("Initializing the server...\n")

  ctx := tag.NewContext(context.Background(),
    tag.Insert(appKey.String(os.Getenv("PROJECT_DOMAIN"))),
    tag.Insert(containerKey.String(os.Getenv("HOSTNAME"))),
	)

	used := diskUsedMetric.Gauge(
		appKey.Value(ctx),
		containerKey.Value(ctx),
	)
  quota := diskQuotaMetric.Gauge(
		appKey.Value(ctx),
		containerKey.Value(ctx),
	)

  go updateDiskMetrics(ctx, &used, &quota)

  err := http.ListenAndServe(":3000", nil)
  if err != nil {
    log.Fatalf("Could not start web server: %s", err)
  }
}
