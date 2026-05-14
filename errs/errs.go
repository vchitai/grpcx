// Package errs provides gRPC-aware error construction and parsing helpers.
// Errors carry a machine-readable Code (i18n key), an English fallback Message,
// and optional structured Metadata — all serialized as google.rpc.ErrorInfo details.
//
// Typical usage in a service handler:
//
//	if !hasStock {
//	    return nil, errs.FailedPrecondition("ERR_OUT_OF_STOCK", "insufficient stock").
//	        WithMetadata("sku", req.Sku).
//	        WithMetadata("available", stock)
//	}
package errs

import (
	"database/sql"
	"fmt"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// defaultDomain is included in ErrorInfo.Domain.
// Set once at startup via SetDomain("myservice.example.com").
var defaultDomain = "grpcx"

// SetDomain configures the error domain reported in gRPC error details.
// Must be called before any concurrent goroutines that call GRPCStatus() are started.
func SetDomain(domain string) { defaultDomain = domain }

// ErrDBNotFound is sql.ErrNoRows, exposed for import-free comparison.
var ErrDBNotFound = sql.ErrNoRows

// Error is a gRPC-aware application error.
// Code is the stable i18n key (e.g. "ERR_INSUFFICIENT_STOCK").
// Message is the English-language developer fallback.
type Error struct {
	StatusCode codes.Code
	Code       string
	Message    string
	Metadata   map[string]any
}

// New creates an Error with the given gRPC status code, error code, and message.
func New(statusCode codes.Code, code string, message string) *Error {
	return &Error{StatusCode: statusCode, Code: code, Message: message}
}

// Errorf creates an Error with a formatted message.
func Errorf(statusCode codes.Code, code string, format string, a ...any) *Error {
	return &Error{StatusCode: statusCode, Code: code, Message: fmt.Sprintf(format, a...)}
}

// Error implements the error interface.
func (e *Error) Error() string { return e.Message }

// WithMetadata attaches a key-value pair to the error for structured context.
func (e *Error) WithMetadata(key string, value any) *Error {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// GRPCStatus converts the error to a gRPC *status.Status with ErrorInfo details attached.
// This satisfies the grpc/status GRPCStatus() interface, so gRPC interceptors handle it automatically.
func (e *Error) GRPCStatus() *status.Status {
	stt := status.New(e.StatusCode, e.Message)
	if s, err := stt.WithDetails(&errdetails.ErrorInfo{
		Reason:   e.Code,
		Domain:   defaultDomain,
		Metadata: toStringMap(e.Metadata),
	}); err == nil {
		return s
	}
	return stt
}

func toStringMap(m map[string]any) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}

// --- Shorthand constructors ---

func NotFound(code, msg string) *Error           { return New(codes.NotFound, code, msg) }
func Unauthenticated(code, msg string) *Error    { return New(codes.Unauthenticated, code, msg) }
func PermissionDenied(code, msg string) *Error   { return New(codes.PermissionDenied, code, msg) }
func AlreadyExists(code, msg string) *Error      { return New(codes.AlreadyExists, code, msg) }
func InvalidArgument(code, msg string) *Error    { return New(codes.InvalidArgument, code, msg) }
func Internal(code, msg string) *Error           { return New(codes.Internal, code, msg) }
func FailedPrecondition(code, msg string) *Error { return New(codes.FailedPrecondition, code, msg) }
func Unavailable(code, msg string) *Error        { return New(codes.Unavailable, code, msg) }

// Internalf creates an Internal error with a formatted message.
func Internalf(code string, format string, a ...any) error {
	return Errorf(codes.Internal, code, format, a...)
}

// Validation creates an InvalidArgument error with field-level violations.
// violations maps field name → stable i18n key (e.g. "sku" → "FIELD_REQUIRED").
// The gateway extracts these into a flat metadata map with code "VALIDATION_ERROR".
func Validation(msg string, violations map[string]string) error {
	stt := status.New(codes.InvalidArgument, msg)
	br := &errdetails.BadRequest{}
	for field, desc := range violations {
		br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
			Field:       field,
			Description: desc,
		})
	}
	if s, err := stt.WithDetails(br); err == nil {
		return s.Err()
	}
	return stt.Err()
}

// Parse extracts the error code and metadata from a gRPC error's ErrorInfo detail.
// Returns empty strings/nil if the error carries no ErrorInfo.
// Used for service-to-service error inspection.
func Parse(err error) (code string, metadata map[string]any) {
	st, ok := status.FromError(err)
	if !ok {
		return "", nil
	}
	for _, detail := range st.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			m := make(map[string]any, len(info.Metadata))
			for k, v := range info.Metadata {
				m[k] = v
			}
			return info.Reason, m
		}
	}
	return "", nil
}
