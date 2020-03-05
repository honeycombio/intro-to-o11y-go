package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/honeycombio/opentelemetry-exporter-go/honeycomb"
	// "github.com/lightstep/opentelemetry-exporter-go/lightstep"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/exporters/trace/stackdriver"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/codes"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
  "go.opentelemetry.io/otel/exporters/metric/prometheus"
	mout "go.opentelemetry.io/otel/exporters/metric/stdout"
	"go.opentelemetry.io/otel/exporters/trace/stdout"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"go.opentelemetry.io/otel/plugin/othttp"
)

func main() {
	serviceName, _ := os.LookupEnv("PROJECT_NAME")

	pusher, err := mout.InstallNewPipeline(mout.Config{
		Quantiles:   []float64{0.5, 0.9, 0.99},
		PrettyPrint: false,
	})
	defer pusher.Stop()

	prom, metricsHandler, err := prometheus.InstallNewPipeline(prometheus.Config{
		DefaultSummaryQuantiles: []float64{0.5, 0.9, 0.99},
	})
	defer prom.Stop()

  
	// stdout exporter
	std, err := stdout.NewExporter(stdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatal(err)
	}

	// honeycomb exporter
	apikey, _ := os.LookupEnv("HNY_KEY")
	dataset, _ := os.LookupEnv("HNY_DATASET")
	hny, err := honeycomb.NewExporter(
		honeycomb.Config{
			APIKey: apikey,
		},
		honeycomb.TargetingDataset(dataset),
		honeycomb.WithServiceName(serviceName))
	if err != nil {
		log.Fatal(err)
	}
	defer hny.Close()

	// Stackdriver exporter
	// Crecential file specified in GOOGLE_APPLICATION_CREDENTIALS in .env is automatically detected.
	sdExporter, err := stackdriver.NewExporter()
	if err != nil {
		log.Fatal(err)
	}

	// jaeger exporter
	jaegerEndpoint, _ := os.LookupEnv("JAEGER_ENDPOINT")
	jExporter, err := jaeger.NewExporter(
		jaeger.WithCollectorEndpoint(jaegerEndpoint),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: serviceName,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	//lExporter, err := lightstep.NewExporter(
	//	lightstep.WithAccessToken(os.Getenv("LS_KEY")),
	//	lightstep.WithServiceName(serviceName))
	//defer lExporter.Close()

	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(std), sdktrace.WithSyncer(hny),
		sdktrace.WithSyncer(jExporter), sdktrace.WithSyncer(sdExporter)) //, sdktrace.WithSyncer(lExporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)

	mux := http.NewServeMux()
	mux.Handle("/", othttp.NewHandler(http.HandlerFunc(rootHandler), "root", othttp.WithSpanOptions(trace.WithNewRoot())))
	mux.Handle("/favicon.ico", http.NotFoundHandler())
	mux.Handle("/fib", othttp.NewHandler(http.HandlerFunc(fibHandler), "fibonacci", othttp.WithSpanOptions(trace.WithNewRoot())))
	mux.Handle("/fibinternal", othttp.NewHandler(http.HandlerFunc(fibHandler), "fibonacci"))
  mux.Handle("/metrics", metricsHandler)
	os.Stderr.WriteString("Initializing the server...\n")

	go updateDiskMetrics(context.Background())

	err = http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatalf("Could not start web server: %s", err)
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	trace.SpanFromContext(ctx).AddEvent(ctx, "annotation within span")
	_ = dbHandler(ctx, "foo")

	fmt.Fprintf(w, "Click [Tools] > [Logs] to see spans!")
}

func fibHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	tr := global.TraceProvider().Tracer("fibHandler")
	var err error
	var i int
	if len(req.URL.Query()["i"]) != 1 {
		err = fmt.Errorf("Wrong number of arguments.")
	} else {
		i, err = strconv.Atoi(req.URL.Query()["i"][0])
	}
	if err != nil {
		fmt.Fprintf(w, "Couldn't parse index '%s'.", req.URL.Query()["i"])
		w.WriteHeader(503)
		return
	}
	trace.SpanFromContext(ctx).SetAttributes(key.Int("parameter", i))
	ret := 0
	failed := false

	if i < 2 {
		ret = 1
	} else {
		// Call /fib?i=(n-1) and /fib?i=(n-2) and add them together.
		var mtx sync.Mutex
		var wg sync.WaitGroup
		client := http.DefaultClient
		for offset := 1; offset < 3; offset++ {
			wg.Add(1)
			go func(n int) {
				err := tr.WithSpan(ctx, "fibClient", func(ictx context.Context) error {
					url := fmt.Sprintf("http://127.0.0.1:3000/fibinternal?i=%d", n)
					trace.SpanFromContext(ictx).SetAttributes(key.String("url", url))
					trace.SpanFromContext(ictx).AddEvent(ictx, "Fib loop count", key.Int("fib-loop", n))
					req, _ := http.NewRequestWithContext(ictx, "GET", url, nil)
					ictx, req = httptrace.W3C(ictx, req)
					httptrace.Inject(ictx, req)
					res, err := client.Do(req)
					if err != nil {
						return err
					}
					body, err := ioutil.ReadAll(res.Body)
					res.Body.Close()
					if err != nil {
						return err
					}
					resp, err := strconv.Atoi(string(body))
					if err != nil {
						return err
					}
					trace.SpanFromContext(ictx).SetStatus(codes.OK)
					trace.SpanFromContext(ictx).SetAttributes(key.Int("result", resp))
					mtx.Lock()
					defer mtx.Unlock()
					ret += resp
					return err
				})
				if err != nil {
					if !failed {
						w.WriteHeader(503)
						failed = true
					}
					fmt.Fprintf(w, "Failed to call child index '%s'.\n", n)
				}
				wg.Done()
			}(i - offset)
		}
		wg.Wait()
	}
	trace.SpanFromContext(ctx).SetAttributes(key.Int("result", ret))
	fmt.Fprintf(w, "%d", ret)
}

func updateDiskMetrics(ctx context.Context) {
	appKey := key.New("glitch.com/app")                // The Glitch app name.
	containerKey := key.New("glitch.com/container_id") // The Glitch container id.

	meter := global.MeterProvider().Meter("container")
	mem := meter.NewInt64Gauge("mem_usage",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of memory used."),
	)
	used := meter.NewFloat64Gauge("disk_usage",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of disk used."),
	)
	quota := meter.NewFloat64Gauge("disk_quota",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of disk quota available."),
	)
	goroutines := meter.NewInt64Gauge("num_goroutines",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of goroutines running."),
	)

	var m runtime.MemStats
	for {
		runtime.ReadMemStats(&m)

		var stat syscall.Statfs_t
		wd, _ := os.Getwd()
		syscall.Statfs(wd, &stat)

		all := float64(stat.Blocks) * float64(stat.Bsize)
		free := float64(stat.Bfree) * float64(stat.Bsize)

		meter.RecordBatch(ctx, meter.Labels(
			appKey.String(os.Getenv("PROJECT_DOMAIN")),
			containerKey.String(os.Getenv("HOSTNAME"))),
			used.Measurement(all-free),
			quota.Measurement(all),
			mem.Measurement(int64(m.Sys)),
			goroutines.Measurement(int64(runtime.NumGoroutine())),
		)
		time.Sleep(time.Minute)
	}
}

func dbHandler(ctx context.Context, color string) int {
	tr := global.TraceProvider().Tracer("dbHandler")
	ctx, span := tr.Start(ctx, "database")
	defer span.End()

	// Pretend we talked to a database here.
	return 0
}
