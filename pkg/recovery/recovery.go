package recovery

import (
	"github.com/cherry1c/arya-frame/pkg/log"
	"github.com/cherry1c/arya-frame/pkg/metric"
	"github.com/go-kit/log/level"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"runtime/debug"
)

var GrpcPanicRecoveryHandler func(p any) (err error)

func Init() error {
	GrpcPanicRecoveryHandler = func(p any) (err error) {
		metric.PanicsTotal.Inc()
		level.Error(log.RpcLogger).Log("msg", "recovered from panic", "panic", p, "stack", debug.Stack())
		return status.Errorf(codes.Internal, "%s", p)
	}

	return nil
}
