package auth

import (
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

var (
	AuthFn        func(ctx context.Context) (context.Context, error)
	AllButHealthZ func(ctx context.Context, callMeta interceptors.CallMeta) bool
)

func Init() error {
	// Setup custom auth.
	AuthFn = func(ctx context.Context) (context.Context, error) {
		token, err := auth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, err
		}
		// TODO: This is example only, perform proper Oauth/OIDC verification!
		if token != "yolo" {
			return nil, status.Error(codes.Unauthenticated, "invalid auth token")
		}
		// NOTE: You can also pass the token in the context for further interceptors or gRPC service code.
		return ctx, nil
	}

	// Setup auth matcher.
	AllButHealthZ = func(ctx context.Context, callMeta interceptors.CallMeta) bool {
		return healthpb.Health_ServiceDesc.ServiceName != callMeta.Service
	}

	return nil
}
