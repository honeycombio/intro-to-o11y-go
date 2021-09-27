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
  
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otlp "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	serviceName, _ := os.LookupEnv("SERVICE_NAME")

	// stdout exporter
	std, err := stdout.NewExporter(stdout.WithPrettyPrint())
	if err != nil {
		log.Fatal(err)
	}

	// honeycomb OTLP gRPC exporter
	apikey, _ := os.LookupEnv("HONEYCOMB_API_KEY")
	dataset, _ := os.LookupEnv("HONEYCOMB_DATASET")
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

	tp := sdktrace.NewTracerProvider(
    sdktrace.WithSampler(sdktrace.AlwaysSample()),
    sdktrace.WithResource(resource.NewWithAttributes(semconv.ServiceNameKey.String(serviceName))),
		sdktrace.WithSyncer(std), 
    sdktrace.WithBatcher(hny)) 
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	mux := http.NewServeMux()
	mux.Handle("/", otelhttp.NewHandler(otelhttp.WithRouteTag("/", http.HandlerFunc(rootHandler)), "root", otelhttp.WithPublicEndpoint()))
  mux.Handle("/sequence.js", otelhttp.NewHandler(otelhttp.WithRouteTag("/sequence.js", http.HandlerFunc(jsHandler)), "sequence-js", otelhttp.WithPublicEndpoint()))
	mux.Handle("/favicon.ico", http.NotFoundHandler())
	mux.Handle("/fib", otelhttp.NewHandler(otelhttp.WithRouteTag("/fib", http.HandlerFunc(fibHandler)), "fibonacci", otelhttp.WithPublicEndpoint()))
	mux.Handle("/fibinternal", otelhttp.NewHandler(otelhttp.WithRouteTag("/fibinternal", http.HandlerFunc(fibHandler)), "fibonacci"))
	os.Stderr.WriteString("Initializing the server...\n")

	err = http.ListenAndServe(":3000", mux)
	if err != nil {
		log.Fatalf("Could not start web server: %s", err)
	}
}

func fibHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	tr := otel.Tracer("fibHandler")
	var err error
	var i int
  var indexParameter = req.URL.Query()["index"]; 
	if len(indexParameter) != 1 {
		err = fmt.Errorf("please pass index as a query parameter")
	} else {
		i, err = strconv.Atoi(indexParameter[0])
	}
	if err != nil {
		fmt.Fprintf(w, "Couldn't parse index '%s'.", indexParameter)
		w.WriteHeader(503)
		return
	}
  
  // add the index parameter as a custom attribute to the current span here
  
	ret := 0
	failed := false

  if i <= 0 {
    ret = 0
  } else if i <= 1 {
		ret = 1
	} else {
		// Call /fib?index=(n-1) and /fib?index=(n-2) and add them together.
		var mtx sync.Mutex
		var wg sync.WaitGroup
		client := http.DefaultClient
		for offset := 1; offset < 3; offset++ {
			wg.Add(1)
			go func(n int) {
				err := func() error {
					ictx, sp := tr.Start(ctx, "fibClient")
					defer sp.End()
					url := fmt.Sprintf("http://127.0.0.1:3000/fibinternal?index=%d", n)
					// trace.SpanFromContext(ictx).SetAttributes(attribute.String("url", url))
					// trace.SpanFromContext(ictx).AddEvent("Fib loop count", trace.WithAttributes(attribute.Int("fib-loop", n)))
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
          
          // here's some exciting addition. Put it in its own span
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


func rootHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	trace.SpanFromContext(ctx).AddEvent("this is an annotation within the span")
  
  var html = `
<html>
  <head>
    <title>Fibonacci Microservice</title>
    <style>
      .fibonacci-sequence {
        margin:20px;
        padding:10px;
        font-family: Monospace;
        font-size:larger;
        border: 1px black solid;
      }
    </style>
       <script src="/sequence.js" defer></script>
 
  </head>
  <body>
    <header>
      <h1>
         A sequence of numbers:
      </h1>
    </header>

    <main>
      <button id="go-button">
        Go
      </button>
      <div id="put-numbers-here" class="fibonacci-sequence">
        &nbsp;
      </div>
      <button id="stop-button">
        Stop
      </button>

    </main>

  </body>
</html>`

	fmt.Fprintf(w, html)
}


func jsHandler(w http.ResponseWriter, req *http.Request) {
  
  var js = `console.log("hello from sequence.js");

const putNumbersHere = document.getElementById("put-numbers-here");
const goButton = document.getElementById("go-button");
const stopButton = document.getElementById("stop-button");
var stopRequested = false;

function formatFibonacciNumber(n) {
  const container = document.createElement("span");

  const numberSpan = document.createElement("span");
  numberSpan.classList.add("fibonacci-number");
  numberSpan.appendChild(document.createTextNode(n));

  const separatorSpan = document.createElement("span");
  numberSpan.classList.add("separator");
  numberSpan.appendChild(document.createTextNode(", "));
  container.appendChild(numberSpan);
  container.appendChild(separatorSpan);
  return container;
}

const unicodeBomb = "\u{1F4A3}";
function indicateError() {
  return document.createTextNode(unicodeBomb);
}

const unicodeEllipsis = "…"
function indicateLoading() {
  const loadingSpan = document.createElement("span");
  loadingSpan.appendChild(document.createTextNode(unicodeEllipsis));
  return loadingSpan;
}

const unicodeStop = "\u{1F6D1}";
function indicateStop() {
  return document.createTextNode(unicodeStop);
}

function addNumbersToSequence(startingIndex) {
  const placeToPutTheNumber = document.createElement("span");
  putNumbersHere.appendChild(placeToPutTheNumber);

  if (stopRequested) {
    placeToPutTheNumber.appendChild(indicateStop());
    console.log("stopping");
    return;
  }

  placeToPutTheNumber.appendChild(indicateLoading());

  const i = startingIndex;
  const url = "/fib?index=" + i;
  fetch(url).then(response => {
    if (response.ok) {
      console.log("ok for " + i);
      response
        .json()
        .then(n => {
          placeToPutTheNumber.replaceChildren(formatFibonacciNumber(n));
          addNumbersToSequence(i + 1);
        }, err => {
          placeToPutTheNumber.replaceChildren(indicateError());
          console.log("parsing error on " + i);
        });
    } else {
      placeToPutTheNumber.replaceChildren(indicateError());
      console.log("error on " + i);
    }
  });
}

function go() {
  stopRequested = false;
  putNumbersHere.replaceChildren();
  addNumbersToSequence(0);
}

goButton.addEventListener("click", go);

function stop() {
  console.log("I hear you. Setting stopRequested");
  stopRequested = true;
}
stopButton.addEventListener("click", stop);

`

	fmt.Fprintf(w, js)
}