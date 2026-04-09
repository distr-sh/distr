package svc

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/tracers"
	sentryotlp "github.com/getsentry/sentry-go/otel/otlp"
	"github.com/go-logr/zapr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

func (r *Registry) GetTracers() *tracers.Tracers {
	return r.tracers
}

func (reg *Registry) createTracer(ctx context.Context) (*tracers.Tracers, error) {
	otel.SetLogger(zapr.NewLogger(reg.logger))

	tpopts := []trace.TracerProviderOption{}
	tmps := []propagation.TextMapPropagator{propagation.TraceContext{}, propagation.Baggage{}}

	if env.OtelExporterOtlpEnabled() {
		if exp, err := otlptracegrpc.New(ctx); err != nil {
			return nil, err
		} else {
			tpopts = append(tpopts, trace.WithSpanProcessor(trace.NewBatchSpanProcessor(exp)))
		}
	}

	if env.OtelExporterSentryEnabled() {
		if exp, err := sentryotlp.NewTraceExporter(ctx, env.SentryDSN()); err != nil {
			return nil, err
		} else {
			tpopts = append(tpopts, trace.WithSpanProcessor(trace.NewBatchSpanProcessor(exp)))
		}
	}

	tracers := tracers.Tracers{
		DefaultProvider: trace.NewTracerProvider(tpopts...),
		AlwaysProvider:  trace.NewTracerProvider(append(tpopts, trace.WithSampler(trace.AlwaysSample()))...),
	}

	otel.SetTracerProvider(tracers.DefaultProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(tmps...))

	if cfg := env.OtelAgentSampler(); cfg != nil {
		tracers.AgentProvider = trace.NewTracerProvider(append(
			tpopts,
			trace.WithSampler(samplerFromConfig(cfg)),
		)...)
	}

	if cfg := env.OtelRegistrySampler(); cfg != nil {
		tracers.RegistryProvider = trace.NewTracerProvider(append(
			tpopts,
			trace.WithSampler(samplerFromConfig(cfg)),
		)...)
	}

	return &tracers, nil
}

func samplerFromConfig(cfg *env.SamplerConfig) trace.Sampler {
	switch cfg.Sampler {
	case env.SamplerAlwaysOn:
		return trace.AlwaysSample()
	case env.SamplerAlwaysOff:
		return trace.NeverSample()
	case env.SamplerTraceIDRatio:
		return trace.TraceIDRatioBased(cfg.Arg)
	case env.SamplerParentBasedAlwaysOn:
		return trace.ParentBased(trace.AlwaysSample())
	case env.SamplerParsedBasedAlwaysOff:
		return trace.ParentBased(trace.NeverSample())
	case env.SamplerParentBasedTraceIDRatio:
		return trace.ParentBased(trace.TraceIDRatioBased(cfg.Arg))
	default:
		panic(fmt.Sprintf("invalid SamplerType: %v", cfg.Sampler))
	}
}
