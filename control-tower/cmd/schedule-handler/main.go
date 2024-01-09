package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/japb1998/control-tower/pkg/sms"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// TODO: change logger to use slog.
var handlerLogger = log.New(os.Stdout, "[Handler] ", log.Default().Flags())
var tracer trace.Tracer
var msgSvc *sms.MsgSvc
var apiUrl = os.Getenv("API_URL")

func main() {
	ctx := context.Background()
	tp, err := xrayconfig.NewTracerProvider(ctx)
	if err != nil {
		fmt.Printf("error creating tracer provider: %v", err)
	}
	defer func(ctx context.Context) {
		err := tp.Shutdown(ctx)
		if err != nil {
			fmt.Printf("error shutting down tracer provider: %v", err)
		}
	}(ctx)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})
	tracer = tp.Tracer("main")
	// we init our routes here
	lambda.Start(otellambda.InstrumentHandler(handler, xrayconfig.WithRecommendedOptions(tp)...))
}
