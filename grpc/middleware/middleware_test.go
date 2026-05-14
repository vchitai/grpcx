package middleware_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/vchitai/grpcx/errs"
	"github.com/vchitai/grpcx/grpc/middleware"
	"github.com/vchitai/grpcx/rpcctx"
)

func unaryInfo(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: method}
}

// --- ErrorWrapperUnaryServerInterceptor ---

func TestErrorWrapper_errsError(t *testing.T) {
	inter := middleware.ErrorWrapperUnaryServerInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errs.NotFound("ERR_NOT_FOUND", "not found")
	}
	_, err := inter(context.Background(), nil, unaryInfo("/svc/Method"), handler)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestErrorWrapper_grpcStatusPassthrough(t *testing.T) {
	inter := middleware.ErrorWrapperUnaryServerInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	}
	_, err := inter(context.Background(), nil, unaryInfo("/svc/Method"), handler)
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.PermissionDenied, st.Code())
}

func TestErrorWrapper_unhandledBecomesInternal(t *testing.T) {
	inter := middleware.ErrorWrapperUnaryServerInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("unexpected db error")
	}
	_, err := inter(context.Background(), nil, unaryInfo("/svc/Method"), handler)
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unknown, st.Code())
}

func TestErrorWrapper_noError(t *testing.T) {
	inter := middleware.ErrorWrapperUnaryServerInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	resp, err := inter(context.Background(), nil, unaryInfo("/svc/Method"), handler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

// --- RequestIDUnaryServerInterceptor ---

func TestRequestID_generated(t *testing.T) {
	inter := middleware.RequestIDUnaryServerInterceptor()
	var capturedID string
	handler := func(ctx context.Context, req any) (any, error) {
		capturedID = rpcctx.GetRequestIDFromContext(ctx)
		return nil, nil
	}
	_, err := inter(
		metadata.NewIncomingContext(context.Background(), metadata.MD{}),
		nil, unaryInfo("/svc/Method"), handler,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, capturedID)
}

func TestRequestID_propagated(t *testing.T) {
	inter := middleware.RequestIDUnaryServerInterceptor()
	inMD := metadata.Pairs(rpcctx.HeaderRequestID, "trace-abc")
	ctx := metadata.NewIncomingContext(context.Background(), inMD)

	var capturedID string
	handler := func(ctx context.Context, req any) (any, error) {
		capturedID = rpcctx.GetRequestIDFromContext(ctx)
		return nil, nil
	}
	_, err := inter(ctx, nil, unaryInfo("/svc/Method"), handler)
	require.NoError(t, err)
	assert.Equal(t, "trace-abc", capturedID)
}
