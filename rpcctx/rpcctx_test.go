package rpcctx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"

	"github.com/vchitai/grpcx/rpcctx"
)

func mdCtx(pairs ...string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(pairs...))
}

func TestGetRequestIDFromContext(t *testing.T) {
	ctx := mdCtx(rpcctx.HeaderRequestID, "req-123")
	assert.Equal(t, "req-123", rpcctx.GetRequestIDFromContext(ctx))
}

func TestGetRequestIDFromContext_missing(t *testing.T) {
	assert.Empty(t, rpcctx.GetRequestIDFromContext(context.Background()))
}

func TestAttachRequestIDRoundtrip(t *testing.T) {
	md := metadata.MD{}
	rpcctx.AttachRequestIDToMD(md, "abc")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	assert.Equal(t, "abc", rpcctx.GetRequestIDFromContext(ctx))
}

func TestAttachRequestIDToIncomingContext(t *testing.T) {
	ctx := rpcctx.AttachRequestIDToIncomingContext(context.Background(), "xyz")
	assert.Equal(t, "xyz", rpcctx.GetRequestIDFromContext(ctx))
}

func TestGetHTTPMethodFromContext(t *testing.T) {
	ctx := mdCtx(rpcctx.MdKeyHTTPMethod, http.MethodPost)
	assert.Equal(t, http.MethodPost, rpcctx.GetHTTPMethodFromContext(ctx))
}

func TestGetLanguageFromContext(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"en", "en"},
		{"vi-VN,vi;q=0.9,en-US;q=0.8", "vi"},
		{"zh-Hans-CN", "zh"},
		{"", ""},
	}
	for _, tt := range tests {
		ctx := mdCtx("accept-language", tt.header)
		if tt.header == "" {
			ctx = context.Background()
		}
		assert.Equal(t, tt.want, rpcctx.GetLanguageFromContext(ctx), "header=%q", tt.header)
	}
}

func TestGetIPAddressFromContext(t *testing.T) {
	ctx := mdCtx(rpcctx.HeaderForwardedFor, "1.2.3.4, 5.6.7.8")
	assert.Equal(t, "1.2.3.4", rpcctx.GetIPAddressFromContext(ctx))
}

func TestReadUserIP_realIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Real-Ip", "10.0.0.1")
	assert.Equal(t, "10.0.0.1", rpcctx.ReadUserIP(r))
}

func TestReadUserIP_forwarded(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.2, 10.0.0.3")
	assert.Equal(t, "10.0.0.2", rpcctx.ReadUserIP(r))
}

func TestReadUserIP_remoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "127.0.0.1:9000"
	assert.Equal(t, "127.0.0.1:9000", rpcctx.ReadUserIP(r))
}
