//go:build local
// +build local

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/japb1998/control-tower/internal/api"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
)

func initProvider() (shutdown func(context.Context) error, err error) {
	fmt.Println("initializing jaeger")
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceName("test-service"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	target := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(target), otlptracegrpc.WithInsecure(), otlptracegrpc.WithDialOption(grpc.WithBlock()))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

func initApp() {
	fmt.Println("init local environment")
	ctx := context.Background()
	shutdownFunc, err := initProvider()
	defer shutdownFunc(ctx)
	if err != nil {
		log.Fatal(err)
	}
	r := api.InitRoutes()

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}

}

func init() {

	fmt.Println("init local local.go")
	err := godotenv.Load(".env", "./control-tower/.env")
	if err != nil {
		log.Fatalf("Error loading env vars: %s", err)
	}

}
