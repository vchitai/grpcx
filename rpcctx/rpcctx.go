// Package rpcctx provides helpers for reading and writing gRPC request metadata.
// All functions work with incoming context metadata (server-side).
package rpcctx

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/vchitai/grpcx/errs"
)

const (
	HeaderRequestID    = "X-Request-Id"
	HeaderForwardedFor = "X-Forwarded-For"
	HeaderUserAgent    = "GRPCGateway-User-Agent"
	MdKeyHTTPMethod    = "X-Http-Method"
)

// ReadUserIP extracts the real client IP from X-Real-Ip, X-Forwarded-For, or RemoteAddr.
func ReadUserIP(r *http.Request) string {
	ip, _, _ := strings.Cut(r.Header.Get("X-Real-Ip"), ",")
	if ip == "" {
		ip = strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return strings.TrimSpace(ip)
}

// GetIPAddressFromContext extracts the client IP from gRPC incoming metadata.
func GetIPAddressFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if vals := md.Get(HeaderForwardedFor); len(vals) > 0 {
		return strings.Split(vals[0], ",")[0]
	}
	return ""
}

// GetRequestIDFromContext returns the request ID from gRPC incoming metadata.
func GetRequestIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return strings.Join(md.Get(HeaderRequestID), "")
}

// AttachRequestIDToMD adds a request ID to gRPC metadata.
func AttachRequestIDToMD(md metadata.MD, requestID string) metadata.MD {
	md.Set(HeaderRequestID, requestID)
	return md
}

// AttachRequestIDToIncomingContext injects a request ID into the incoming gRPC context.
func AttachRequestIDToIncomingContext(ctx context.Context, requestID string) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	return metadata.NewIncomingContext(ctx, AttachRequestIDToMD(md, requestID))
}

// GetHTTPMethodFromContext returns the originating HTTP method from gRPC incoming metadata.
func GetHTTPMethodFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return strings.Join(md.Get(MdKeyHTTPMethod), "")
}

// AttachHTTPMethodToMD stores the HTTP method in gRPC metadata.
func AttachHTTPMethodToMD(md metadata.MD, method string) metadata.MD {
	md.Set(MdKeyHTTPMethod, method)
	return md
}

// AttachHTTPMethodToIncomingContext injects the HTTP method into the incoming gRPC context.
func AttachHTTPMethodToIncomingContext(ctx context.Context, method string) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	return metadata.NewIncomingContext(ctx, AttachHTTPMethodToMD(md, method))
}

// GetTokenFromContext extracts the Bearer token from the authorization metadata.
func GetTokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errs.Unauthenticated("MISSING_ACCESS_TOKEN", "missing access token")
	}
	authorization := md.Get("authorization")
	if len(authorization) == 0 {
		return "", errs.Unauthenticated("MISSING_ACCESS_TOKEN", "missing access token")
	}
	return strings.TrimPrefix(authorization[0], "Bearer "), nil
}

// GetPatternFromContext returns the matched HTTP path pattern from gRPC metadata.
func GetPatternFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return strings.Join(md.Get("pattern"), "")
}

// GetUserAgentFromContext returns the User-Agent forwarded by grpc-gateway.
func GetUserAgentFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	return strings.Join(md.Get("grpcgateway-user-agent"), "")
}

// GetMethodFromContext returns the gRPC method name (without the service prefix).
func GetMethodFromContext(ctx context.Context) string {
	method, ok := grpc.Method(ctx)
	if !ok {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(method, "/"), "/")
	if len(parts) < 2 {
		return method
	}
	return parts[1]
}

// GetLanguageFromContext returns the preferred language from the Accept-Language metadata.
// Returns the primary language tag (e.g. "en", "vi"). Falls back to empty string.
func GetLanguageFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get("accept-language")
	if len(vals) == 0 {
		return ""
	}
	// "vi-VN,vi;q=0.9,en-US;q=0.8" → "vi"
	s := strings.SplitN(vals[0], ",", 2)[0]
	s = strings.SplitN(s, ";", 2)[0]
	s = strings.SplitN(s, "-", 2)[0]
	return strings.ToLower(strings.TrimSpace(s))
}
