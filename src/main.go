package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/honeycombio/opentelemetry-exporter-go/honeycomb"
	"go.opentelemetry.io/otel/exporter/trace/jaeger"
	"go.opentelemetry.io/otel/exporter/trace/stackdriver"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/codes"

	"go.opentelemetry.io/otel/api/distributedcontext"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporter/trace/stdout"
	"go.opentelemetry.io/otel/global"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"go.opentelemetry.io/otel/plugin/othttp"
)

var (
	appKey         = key.New("honeycomb.io/glitch/app")          // The Glitch app name.
	containerKey   = key.New("honeycomb.io/glitch/container_id") // The Glitch container id.
	meter          = global.MeterProvider().GetMeter("main")
	diskUsedMetric = meter.NewFloat64Gauge("honeycomb.io/glitch/disk_usage",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of disk used."),
	)
	diskQuotaMetric = meter.NewFloat64Gauge("honeycomb.io/glitch/disk_quota",
		metric.WithKeys(appKey, containerKey),
		metric.WithDescription("Amount of disk quota available."),
	)
)

func main() {
	serviceName, _ := os.LookupEnv("PROJECT_NAME")

	// stdout exporter
	std, err := stdout.NewExporter(stdout.Options{PrettyPrint: true})
	if err != nil {
		log.Fatal(err)
	}

	// honeycomb exporter
	apikey, _ := os.LookupEnv("HNY_KEY")
	dataset, _ := os.LookupEnv("HNY_DATASET")
	hny, err := honeycomb.NewExporter(honeycomb.Config{
		ApiKey:      apikey,
		Dataset:     dataset,
		Debug:       false,
		ServiceName: serviceName,
	})
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

	tp, err := sdktrace.NewProvider(sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(std), sdktrace.WithSyncer(hny),
		sdktrace.WithSyncer(jExporter), sdktrace.WithSyncer(sdExporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)

	mux := http.NewServeMux()
	mux.Handle("/", othttp.NewHandler(http.HandlerFunc(rootHandler), "root"))
	mux.Handle("/favicon.ico", http.NotFoundHandler())
	mux.Handle("/fib", othttp.NewHandler(http.HandlerFunc(fibHandler), "fibonacci", othttp.WithPublicEndpoint()))
	mux.Handle("/fibinternal", othttp.NewHandler(http.HandlerFunc(fibHandler), "fibonacci"))
	mux.Handle("/quitquitquit", http.HandlerFunc(restartHandler))
	os.Stderr.WriteString("Initializing the server...\n")

	ctx := distributedcontext.NewContext(context.Background(),
		appKey.String(os.Getenv("PROJECT_DOMAIN")),
		containerKey.String(os.Getenv("HOSTNAME")),
	)

	commonLabels := meter.Labels(appKey.Int(10))

	used := diskUsedMetric.AcquireHandle(commonLabels)
	quota := diskQuotaMetric.AcquireHandle(commonLabels)

	go updateDiskMetrics(ctx, used, quota)

	err = http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatalf("Could not start web server: %s", err)
	}
}

func rootHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	trace.CurrentSpan(ctx).AddEvent(ctx, "annotation within span")
	_ = dbHandler(ctx, "foo")

	fmt.Fprintf(w, "Click [Tools] > [Logs] to see spans!")
}

func fibHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	tr := global.TraceProvider().GetTracer("fibHandler")
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
	trace.CurrentSpan(ctx).SetAttribute(key.Int("parameter", i))
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
					trace.CurrentSpan(ictx).SetAttributes(key.String("url", url))
					trace.CurrentSpan(ictx).AddEvent(ictx, "Fib loop count", key.Int("fib-loop", n))
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
					trace.CurrentSpan(ictx).SetStatus(codes.OK)
					trace.CurrentSpan(ictx).SetAttributes(key.Int("result", resp))
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
	trace.CurrentSpan(ctx).SetAttribute(key.Int("result", ret))
	fmt.Fprintf(w, "%d", ret)
}

func updateDiskMetrics(ctx context.Context, used, quota metric.Float64GaugeHandle) {
	for {
		var stat syscall.Statfs_t
		wd, _ := os.Getwd()
		syscall.Statfs(wd, &stat)

		all := float64(stat.Blocks) * float64(stat.Bsize)
		free := float64(stat.Bfree) * float64(stat.Bsize)
		used.Set(ctx, all-free)
		quota.Set(ctx, all)
		time.Sleep(time.Minute)
	}
}

func dbHandler(ctx context.Context, color string) int {
	tr := global.TraceProvider().GetTracer("dbHandler")
	ctx, span := tr.Start(ctx, "database")
	defer span.End()

	// Pretend we talked to a database here.
	return 0
}

func restartHandler(w http.ResponseWriter, req *http.Request) {
	os.Exit(0)
}
