// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stdout_test

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName    = "github.com/instrumentron"
	instrumentationVersion = "v0.1.0"
)

var (
	tracer = otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
		trace.WithSchemaURL("https://opentelemetry.io/schemas/1.2.0"),
	)

	meter = global.GetMeterProvider().Meter(
		instrumentationName,
		metric.WithInstrumentationVersion(instrumentationVersion),
	)

	loopCounter = metric.Must(meter).NewInt64Counter("function.loops")
	paramValue  = metric.Must(meter).NewInt64ValueRecorder("function.param")

	nameKey = attribute.Key("function.name")
)

func add(ctx context.Context, x, y int64) int64 {
	nameKV := nameKey.String("add")

	var span trace.Span
	ctx, span = tracer.Start(ctx, "Addition")
	defer span.End()

	loopCounter.Add(ctx, 1, nameKV)
	paramValue.Record(ctx, x, nameKV)
	paramValue.Record(ctx, y, nameKV)

	return x + y
}

func multiply(ctx context.Context, x, y int64) int64 {
	nameKV := nameKey.String("multiply")

	var span trace.Span
	ctx, span = tracer.Start(ctx, "Multiplication")
	defer span.End()

	loopCounter.Add(ctx, 1, nameKV)
	paramValue.Record(ctx, x, nameKV)
	paramValue.Record(ctx, y, nameKV)

	return x * y
}

func Example() {
	exportOpts := []stdout.Option{
		stdout.WithPrettyPrint(),
	}
	// Registers both a trace and meter Provider globally.
	tracerProvider, pusher, err := stdout.InstallNewPipeline(exportOpts, nil)
	if err != nil {
		log.Fatal("Could not initialize stdout exporter:", err)
	}
	ctx := context.Background()

	log.Println("the answer is", add(ctx, multiply(ctx, multiply(ctx, 2, 2), 10), 2))

	if err := pusher.Stop(ctx); err != nil {
		log.Fatal("Could not stop stdout exporter:", err)
	}
	if err := tracerProvider.Shutdown(ctx); err != nil {
		log.Fatal("Could not stop stdout tracer:", err)
	}
}
