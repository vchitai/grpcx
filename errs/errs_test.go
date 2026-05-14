package errs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vchitai/grpcx/errs"
)

func TestNew(t *testing.T) {
	e := errs.New(codes.NotFound, "ERR_NOT_FOUND", "not found")
	assert.Equal(t, codes.NotFound, e.StatusCode)
	assert.Equal(t, "ERR_NOT_FOUND", e.Code)
	assert.Equal(t, "not found", e.Message)
	assert.Equal(t, "not found", e.Error())
}

func TestErrorf(t *testing.T) {
	e := errs.Errorf(codes.Internal, "ERR_INTERNAL", "failed: %s", "timeout")
	assert.Equal(t, "failed: timeout", e.Message)
}

func TestWithMetadata(t *testing.T) {
	e := errs.New(codes.InvalidArgument, "ERR_BAD", "bad").
		WithMetadata("field", "email").
		WithMetadata("count", 3)
	assert.Equal(t, "email", e.Metadata["field"])
	assert.Equal(t, 3, e.Metadata["count"])
}

func TestGRPCStatus_roundtrip(t *testing.T) {
	e := errs.NotFound("ERR_USER_NOT_FOUND", "user not found").
		WithMetadata("id", "abc123")

	st := e.GRPCStatus()
	require.NotNil(t, st)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "user not found", st.Message())

	code, metadata := errs.Parse(st.Err())
	assert.Equal(t, "ERR_USER_NOT_FOUND", code)
	assert.Equal(t, "abc123", metadata["id"])
}

func TestGRPCStatus_noMetadata(t *testing.T) {
	e := errs.Internal("ERR_INTERNAL", "oops")
	code, metadata := errs.Parse(e.GRPCStatus().Err())
	assert.Equal(t, "ERR_INTERNAL", code)
	assert.Empty(t, metadata)
}

func TestParse_nonGRPCError(t *testing.T) {
	code, metadata := errs.Parse(assert.AnError)
	assert.Empty(t, code)
	assert.Nil(t, metadata)
}

func TestParse_noDetails(t *testing.T) {
	st := status.New(codes.Internal, "bare error")
	code, metadata := errs.Parse(st.Err())
	assert.Empty(t, code)
	assert.Nil(t, metadata)
}

func TestValidation(t *testing.T) {
	err := errs.Validation("invalid request", map[string]string{
		"email": "FIELD_INVALID_EMAIL",
		"name":  "FIELD_REQUIRED",
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestShorthands(t *testing.T) {
	tests := []struct {
		name string
		err  *errs.Error
		code codes.Code
	}{
		{"NotFound", errs.NotFound("C", "m"), codes.NotFound},
		{"Unauthenticated", errs.Unauthenticated("C", "m"), codes.Unauthenticated},
		{"PermissionDenied", errs.PermissionDenied("C", "m"), codes.PermissionDenied},
		{"AlreadyExists", errs.AlreadyExists("C", "m"), codes.AlreadyExists},
		{"InvalidArgument", errs.InvalidArgument("C", "m"), codes.InvalidArgument},
		{"Internal", errs.Internal("C", "m"), codes.Internal},
		{"FailedPrecondition", errs.FailedPrecondition("C", "m"), codes.FailedPrecondition},
		{"Unavailable", errs.Unavailable("C", "m"), codes.Unavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.err.StatusCode)
		})
	}
}

func TestSetDomain(t *testing.T) {
	errs.SetDomain("myservice.example.com")
	t.Cleanup(func() { errs.SetDomain("grpcx") })

	e := errs.Internal("ERR_X", "x")
	st := e.GRPCStatus()
	// domain is embedded in ErrorInfo details; parse just checks code round-trips
	code, _ := errs.Parse(st.Err())
	assert.Equal(t, "ERR_X", code)
}
