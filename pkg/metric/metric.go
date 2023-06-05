package metric

import (
	"context"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/trace"
)

var (
	SrvMetrics          *grpcprom.ServerMetrics
	ExemplarFromContext func(ctx context.Context) prometheus.Labels
	PanicsTotal         prometheus.Counter
	Reg                 *prometheus.Registry
)

func Init() error {
	// Setup metrics.
	SrvMetrics = grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	Reg = prometheus.NewRegistry()
	Reg.MustRegister(SrvMetrics)

	ExemplarFromContext = func(ctx context.Context) prometheus.Labels {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return prometheus.Labels{"traceID": span.TraceID().String()}
		}
		return nil
	}

	// Setup metric for panic recoveries.
	PanicsTotal = promauto.With(Reg).NewCounter(prometheus.CounterOpts{
		Name: "grpc_req_panics_recovered_total",
		Help: "Total number of gRPC requests recovered from internal panic.",
	})

	return nil
}
