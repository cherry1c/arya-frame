package log

import (
	"github.com/go-kit/log"
	"os"
)

const component = "grpc-example"

var (
	Logger    log.Logger
	RpcLogger log.Logger
)

func Init() error {
	// Setup logging.
	Logger = log.NewLogfmtLogger(os.Stderr)
	RpcLogger = log.With(Logger, "service", "gRPC/server", "component", component)

	return nil
}
