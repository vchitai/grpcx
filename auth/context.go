package auth

import (
	"context"
	"slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey int

const (
	claimsKey contextKey = iota
	vendorKey
)

// WithClaims attaches JWT Claims to the context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext extracts JWT Claims from the context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok && c != nil
}

// WithVendorIdentity attaches a VendorIdentity to the context.
func WithVendorIdentity(ctx context.Context, v *VendorIdentity) context.Context {
	return context.WithValue(ctx, vendorKey, v)
}

// VendorIdentityFromContext extracts the VendorIdentity from the context.
func VendorIdentityFromContext(ctx context.Context) (*VendorIdentity, bool) {
	v, ok := ctx.Value(vendorKey).(*VendorIdentity)
	return v, ok
}

// RequireAdmin returns PermissionDenied unless the caller has "operator" or "admin" in Roles.
func RequireAdmin(ctx context.Context) error {
	c, ok := ClaimsFromContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if slices.Contains(c.Roles, "operator") || slices.Contains(c.Roles, "admin") {
		return nil
	}
	return status.Errorf(codes.PermissionDenied, "admin role required")
}

// RequireSuperAdmin returns PermissionDenied unless the caller has "admin" in Roles.
func RequireSuperAdmin(ctx context.Context) error {
	c, ok := ClaimsFromContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if slices.Contains(c.Roles, "admin") {
		return nil
	}
	return status.Errorf(codes.PermissionDenied, "super_admin role required")
}
