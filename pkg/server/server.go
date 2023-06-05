package server

import (
	"context"
	"fmt"
	inauth "framework/pkg/auth"
	inlog "framework/pkg/log"
	"framework/pkg/metric"
	inrecovery "framework/pkg/recovery"
	"framework/pkg/trace"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/testing/testpb"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"syscall"
)

func interceptorLogger(l log.Logger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		largs := append([]any{"msg", msg}, fields...)
		switch lvl {
		case logging.LevelDebug:
			_ = level.Debug(l).Log(largs...)
		case logging.LevelInfo:
			_ = level.Info(l).Log(largs...)
		case logging.LevelWarn:
			_ = level.Warn(l).Log(largs...)
		case logging.LevelError:
			_ = level.Error(l).Log(largs...)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}

const (
	component = "grpc-example"
	grpcAddr  = ":8080"
	httpAddr  = ":8081"
)

var grpcSrv *grpc.Server

func Init() error {
	grpcSrv = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// Order matters e.g. tracing interceptor have to create span first for the later exemplars to work.
			otelgrpc.UnaryServerInterceptor(),
			metric.SrvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(metric.ExemplarFromContext)),
			logging.UnaryServerInterceptor(interceptorLogger(inlog.RpcLogger), logging.WithFieldsFromContext(trace.LogTraceID)),
			selector.UnaryServerInterceptor(auth.UnaryServerInterceptor(inauth.AuthFn), selector.MatchFunc(inauth.AllButHealthZ)),
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(inrecovery.GrpcPanicRecoveryHandler)),
		),
		grpc.ChainStreamInterceptor(
			otelgrpc.StreamServerInterceptor(),
			metric.SrvMetrics.StreamServerInterceptor(grpcprom.WithExemplarFromContext(metric.ExemplarFromContext)),
			logging.StreamServerInterceptor(interceptorLogger(inlog.RpcLogger), logging.WithFieldsFromContext(trace.LogTraceID)),
			selector.StreamServerInterceptor(auth.StreamServerInterceptor(inauth.AuthFn), selector.MatchFunc(inauth.AllButHealthZ)),
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(inrecovery.GrpcPanicRecoveryHandler)),
		),
	)
	t := &testpb.TestPingService{}
	testpb.RegisterTestServiceServer(grpcSrv, t)
	metric.SrvMetrics.InitializeMetrics(grpcSrv)

	return nil
}

func Run() error {
	g := &run.Group{}
	g.Add(func() error {
		l, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			return err
		}
		level.Info(inlog.Logger).Log("msg", "starting gRPC server", "addr", l.Addr().String())
		return grpcSrv.Serve(l)
	}, func(err error) {
		grpcSrv.GracefulStop()
		grpcSrv.Stop()
	})

	httpSrv := &http.Server{Addr: httpAddr}
	g.Add(func() error {
		m := http.NewServeMux()
		// Create HTTP handler for Prometheus metrics.
		m.Handle("/metrics", promhttp.HandlerFor(
			metric.Reg,
			promhttp.HandlerOpts{
				// Opt into OpenMetrics e.g. to support exemplars.
				EnableOpenMetrics: true,
			},
		))
		httpSrv.Handler = m
		level.Info(inlog.Logger).Log("msg", "starting HTTP server", "addr", httpSrv.Addr)
		return httpSrv.ListenAndServe()
	}, func(error) {
		if err := httpSrv.Close(); err != nil {
			level.Error(inlog.Logger).Log("msg", "failed to stop web server", "err", err)
		}
	})

	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	if err := g.Run(); err != nil {
		level.Error(inlog.Logger).Log("err", err)
		os.Exit(1)
	}

	return nil
}
