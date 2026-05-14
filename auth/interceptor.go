package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/vchitai/grpcx/errs"
)

// AuthMode describes what authentication a route requires.
type AuthMode int

const (
	AuthModePublic AuthMode = iota // no authentication required
	AuthModeJWT                    // valid JWT required; Claims are injected into context
	AuthModeAPIKey                 // API key required; VendorIdentity is injected into context
)

// RoutePolicy maps a gRPC full method name to its AuthMode.
// Return AuthModePublic for health checks or other unauthenticated endpoints.
//
//	srv, _ := server.New(
//	    server.WithGrpcServerUnaryInterceptors(
//	        auth.NewAuthInterceptor(jwtVal, apiKeyVal, func(method string) auth.AuthMode {
//	            if strings.HasPrefix(method, "/api.health.v1.") {
//	                return auth.AuthModePublic
//	            }
//	            return auth.AuthModeJWT
//	        }),
//	    ),
//	)
type RoutePolicy func(fullMethod string) AuthMode

// NewAuthInterceptor returns a unary interceptor that authenticates each request
// according to policy. Role-level authorization is handled separately in handlers
// using RequireAdmin or RequireSuperAdmin.
func NewAuthInterceptor(jwtVal *JWTValidator, apiKeyVal *APIKeyValidator, policy RoutePolicy) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		switch policy(info.FullMethod) {
		case AuthModeJWT:
			claims, err := extractJWT(ctx, jwtVal)
			if err != nil {
				return nil, err
			}
			ctx = WithClaims(ctx, claims)
		case AuthModeAPIKey:
			vendor, err := extractAPIKey(ctx, apiKeyVal)
			if err != nil {
				return nil, err
			}
			ctx = WithVendorIdentity(ctx, vendor)
		}
		return handler(ctx, req)
	}
}

func extractJWT(ctx context.Context, v *JWTValidator) (*Claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errs.Unauthenticated("MISSING_ACCESS_TOKEN", "missing access token")
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return nil, errs.Unauthenticated("MISSING_ACCESS_TOKEN", "missing access token")
	}
	token := strings.TrimPrefix(vals[0], "Bearer ")
	return v.Validate(token)
}

func extractAPIKey(ctx context.Context, v *APIKeyValidator) (*VendorIdentity, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errs.Unauthenticated("MISSING_API_KEY", "missing api key")
	}
	vals := md.Get("x-api-key")
	if len(vals) == 0 || vals[0] == "" {
		return nil, errs.Unauthenticated("MISSING_API_KEY", "missing api key")
	}
	return v.Validate(vals[0])
}
