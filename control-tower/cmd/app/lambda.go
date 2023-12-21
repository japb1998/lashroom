//go:build !local
// +build !local

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/japb1998/control-tower/internal/api"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
)

func initProvider() (shutdown func(context.Context) error, err error) {
	fmt.Println("initializing xray")
	ctx := context.Background()
	tp, err := xrayconfig.NewTracerProvider(ctx)
	if err != nil {
		return nil, err
	}
	shutdown = func(ctx context.Context) error {
		err := tp.Shutdown(ctx)
		if err != nil {
			return err
		}
		return nil
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	return shutdown, nil
}

func initApp() {
	ctx := context.Background()
	tp, err := xrayconfig.NewTracerProvider(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func(ctx context.Context) {
		err := tp.Shutdown(ctx)
		if err != nil {
			fmt.Printf("error shutting down tracer provider: %v", err)
		}
	}(ctx)
	r := api.InitRoutes()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})
	lambda.Start(otellambda.InstrumentHandler(api.HandlerFunc(r), xrayconfig.WithRecommendedOptions(tp)...))
}
