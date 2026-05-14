package auth

import (
	"context"

	"github.com/vchitai/grpcx/errs"
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
	return c, ok
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

// RequireAdmin returns PermissionDenied if the caller is not admin or super_admin.
func RequireAdmin(ctx context.Context) error {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return errs.Unauthenticated("MISSING_CLAIMS", "missing auth claims")
	}
	if claims.Role != RoleAdmin && claims.Role != RoleSuperAdmin {
		return errs.PermissionDenied("INSUFFICIENT_ROLE", "admin access required")
	}
	return nil
}

// RequireSuperAdmin returns PermissionDenied if the caller is not super_admin.
func RequireSuperAdmin(ctx context.Context) error {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return errs.Unauthenticated("MISSING_CLAIMS", "missing auth claims")
	}
	if claims.Role != RoleSuperAdmin {
		return errs.PermissionDenied("INSUFFICIENT_ROLE", "super admin access required")
	}
	return nil
}
