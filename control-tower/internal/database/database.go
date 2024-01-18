package database

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func getTracer() trace.Tracer {
	if tracer != nil {
		return tracer
	}

	tracer = otel.Tracer("github.com/japb1998/internal/database")
	return tracer
}
