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

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/metric"
	mglobal "go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	serviceName, _ := os.LookupEnv("PROJECT_NAME")

	//pusher, err := mout.InstallNewPipeline(mout.Config{
	//	Quantiles:   []float64{0.5, 0.9, 0.99},
	//	PrettyPrint: false,
	//})
	//defer pusher.Stop()

	prom, err := prometheus.InstallNewPipeline(prometheus.Config{
		DefaultHistogramBoundaries: []float64{0.5, 0.9, 0.99},
	})

	// stdout exporter
	std, err := stdout.NewExporter(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	// honeycomb OTLP gRPC exporter
	apikey, _ := os.LookupEnv("HNY_KEY")
	dataset, _ := os.LookupEnv("HNY_DATASET")
	driver := otlpgrpc.NewDriver(
		otlpgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
		otlpgrpc.WithEndpoint("api.honeycomb.io:443"),
		otlpgrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    apikey,
			"x-honeycomb-dataset": dataset,
		}),
	)
	hny, err := otlp.NewExporter(context.Background(), driver)
	if err != nil {
		log.Fatal(err)
	}
	defer hny.Shutdown(context.Background())

	// jaeger exporter
	jaegerEndpoint, _ := os.LookupEnv("JAEGER_ENDPOINT")
	jExporter, err := jaeger.NewRawExporter(
		jaeger.WithCollectorEndpoint(jaegerEndpoint),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: serviceName,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(std), sdktrace.WithSyncer(hny),
		sdktrace.WithSyncer(jExporter)) // , sdktrace.WithSyncer(sdExporter))
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	mux := http.NewServeMux()
	mux.Handle("/", otelhttp.NewHandler(otelhttp.WithRouteTag("/", http.HandlerFunc(rootHandler)), "root", otelhttp.WithPublicEndpoint()))
	mux.Handle("/favicon.ico", http.NotFoundHandler())
	mux.Handle("/fib", otelhttp.NewHandler(otelhttp.WithRouteTag("/fib", http.HandlerFunc(fibHandler)), "fibonacci", otelhttp.WithPublicEndpoint()))
	mux.Handle("/fibinternal", otelhttp.NewHandler(otelhttp.WithRouteTag("/fibinternal", http.HandlerFunc(fibHandler)), "fibonacci"))
	mux.Handle("/metrics", prom)
	os.Stderr.WriteString("Initializing the server...\n")

	go updateDiskMetrics(context.Background())

	err = http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatalf("Could not start web server: %s", err)
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	trace.SpanFromContext(ctx).AddEvent("annotation within span")
	_ = dbHandler(ctx, "foo")

	fmt.Fprintf(w, "Click [Tools] > [Logs] to see spans!")
}

func fibHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	tr := otel.Tracer("fibHandler")
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
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int("parameter", i))
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
				err := func() error {
					ictx, sp := tr.Start(ctx, "fibClient")
					defer sp.End()
					url := fmt.Sprintf("http://127.0.0.1:3000/fibinternal?i=%d", n)
					trace.SpanFromContext(ictx).SetAttributes(attribute.String("url", url))
					trace.SpanFromContext(ictx).AddEvent("Fib loop count", trace.WithAttributes(attribute.Int("fib-loop", n)))
					req, _ := http.NewRequestWithContext(ictx, "GET", url, nil)
					ictx, req = otelhttptrace.W3C(ictx, req)
					otelhttptrace.Inject(ictx, req)
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
						trace.SpanFromContext(ictx).SetStatus(codes.Error, "failure parsing")
						return err
					}
					trace.SpanFromContext(ictx).SetAttributes(attribute.Int("result", resp))
					mtx.Lock()
					defer mtx.Unlock()
					ret += resp
					return err
				}()
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
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int("result", ret))
	fmt.Fprintf(w, "%d", ret)
}

func updateDiskMetrics(ctx context.Context) {
	appKey := attribute.Key("glitch.com/app")                // The Glitch app name.
	containerKey := attribute.Key("glitch.com/container_id") // The Glitch container id.

	meter := mglobal.Meter("container")
	mem, _ := meter.NewInt64ValueRecorder("mem_usage",
		metric.WithDescription("Amount of memory used."),
	)
	used, _ := meter.NewFloat64ValueRecorder("disk_usage",
		metric.WithDescription("Amount of disk used."),
	)
	quota, _ := meter.NewFloat64ValueRecorder("disk_quota",
		metric.WithDescription("Amount of disk quota available."),
	)
	goroutines, _ := meter.NewInt64ValueRecorder("num_goroutines",
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

		meter.RecordBatch(ctx, []attribute.KeyValue{
			appKey.String(os.Getenv("PROJECT_DOMAIN")),
			containerKey.String(os.Getenv("HOSTNAME"))},
			used.Measurement(all-free),
			quota.Measurement(all),
			mem.Measurement(int64(m.Sys)),
			goroutines.Measurement(int64(runtime.NumGoroutine())),
		)
		time.Sleep(time.Minute)
	}
}

func dbHandler(ctx context.Context, color string) int {
	tr := otel.Tracer("dbHandler")
	ctx, span := tr.Start(ctx, "database")
	defer span.End()

	// Pretend we talked to a database here.
	return 0
}
