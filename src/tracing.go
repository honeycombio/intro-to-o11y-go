package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	otlp "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func InitializeTracing(ctx context.Context) *otlp.Exporter {

	// stdout exporter
	// uncomment below (four lines here, and one on line ~53) to see your events printed to the console
	// std, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	// if err != nil {
	//   log.Fatal(err)
	// }

	// honeycomb OTLP gRPC exporter
	apikey, _ := os.LookupEnv("HONEYCOMB_API_KEY")
	serviceName, _ := os.LookupEnv("SERVICE_NAME")
	os.Stderr.WriteString(fmt.Sprintf("Sending to Honeycomb with API Key <%s> and service name %s\n", apikey, serviceName))

	// set up grpc
	driver := otlpgrpc.NewClient(
		otlpgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
		otlpgrpc.WithEndpoint("api.honeycomb.io:443"),
		otlpgrpc.WithHeaders(map[string]string{
			"x-honeycomb-team": apikey,
		}),
	)
	hny, err := otlp.New(ctx, driver)
	if err != nil {
		log.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(serviceName))),
		// uncomment (one line here, plus four above) to see your events printed to the console
		// sdktrace.WithSyncer(std),
		sdktrace.WithBatcher(hny))
	if err != nil {
		log.Fatal(err)
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return hny
}
