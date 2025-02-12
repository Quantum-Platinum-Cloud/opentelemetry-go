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

package otlptrace // import "go.opentelemetry.io/otel/exporters/otlp/otlptrace"

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/internal/tracetransform"

	"go.opentelemetry.io/otel"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

var (
	errAlreadyStarted = errors.New("already started")
)

// Exporter exports trace data in the OTLP wire format.
type Exporter struct {
	client Client

	mu      sync.RWMutex
	started bool

	startOnce sync.Once
	stopOnce  sync.Once
}

// ExportSpans exports a batch of spans.
func (e *Exporter) ExportSpans(ctx context.Context, ss []tracesdk.ReadOnlySpan) error {
	protoSpans := tracetransform.Spans(ss)
	if len(protoSpans) == 0 {
		return nil
	}

	return e.client.UploadTraces(ctx, protoSpans)
}

// Start establishes a connection to the receiving endpoint.
func (e *Exporter) Start(ctx context.Context) error {
	var err = errAlreadyStarted
	e.startOnce.Do(func() {
		e.mu.Lock()
		e.started = true
		e.mu.Unlock()
		err = e.client.Start(ctx)
	})

	return err
}

// Shutdown flushes all exports and closes all connections to the receiving endpoint.
func (e *Exporter) Shutdown(ctx context.Context) error {
	e.mu.RLock()
	started := e.started
	e.mu.RUnlock()

	if !started {
		return nil
	}

	var err error

	e.stopOnce.Do(func() {
		err = e.client.Stop(ctx)
		e.mu.Lock()
		e.started = false
		e.mu.Unlock()
	})

	return err
}

var _ tracesdk.SpanExporter = (*Exporter)(nil)

// NewExporter constructs a new Exporter and starts it.
func NewExporter(ctx context.Context, client Client) (*Exporter, error) {
	exp := NewUnstartedExporter(client)
	if err := exp.Start(ctx); err != nil {
		return nil, err
	}
	return exp, nil
}

// NewUnstartedExporter constructs a new Exporter and does not start it.
func NewUnstartedExporter(client Client) *Exporter {
	return &Exporter{
		client: client,
	}
}

// NewExportPipeline sets up a complete export pipeline
// with the recommended TracerProvider setup.
func NewExportPipeline(ctx context.Context, client Client) (*Exporter, *tracesdk.TracerProvider, error) {
	exp, err := NewExporter(ctx, client)
	if err != nil {
		return nil, nil, err
	}

	tracerProvider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
	)

	return exp, tracerProvider, nil
}

// InstallNewPipeline instantiates a NewExportPipeline with the
// recommended configuration and registers it globally.
func InstallNewPipeline(ctx context.Context, client Client) (*Exporter, *tracesdk.TracerProvider, error) {
	exp, tp, err := NewExportPipeline(ctx, client)
	if err != nil {
		return nil, nil, err
	}

	otel.SetTracerProvider(tp)
	return exp, tp, err
}
