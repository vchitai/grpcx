package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/vchitai/grpcx/auth"
	"github.com/vchitai/grpcx/errs"
)

const testSecret = "test-secret-key"

func makeToken(t *testing.T, userID string, role auth.Role, secret string) string {
	t.Helper()
	type claims struct {
		jwt.RegisteredClaims
		UserID string    `json:"user_id"`
		Role   auth.Role `json:"role"`
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		UserID:           userID,
		Role:             role,
	})
	s, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

func ctxWithAuth(token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewIncomingContext(context.Background(), md)
}

func ctxWithAPIKey(key string) context.Context {
	md := metadata.Pairs("x-api-key", key)
	return metadata.NewIncomingContext(context.Background(), md)
}

// --- JWTValidator ---

func TestJWTValidator_valid(t *testing.T) {
	v := auth.NewJWTValidator(testSecret)
	tok := makeToken(t, "user-1", auth.RoleUser, testSecret)
	claims, err := v.Validate(tok)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, auth.RoleUser, claims.Role)
}

func TestJWTValidator_wrongSecret(t *testing.T) {
	v := auth.NewJWTValidator(testSecret)
	tok := makeToken(t, "user-1", auth.RoleUser, "wrong-secret")
	_, err := v.Validate(tok)
	require.Error(t, err)
	code, _ := errs.Parse(err)
	assert.Equal(t, "INVALID_TOKEN", code)
}

func TestJWTValidator_expired(t *testing.T) {
	type claims struct {
		jwt.RegisteredClaims
		UserID string    `json:"user_id"`
		Role   auth.Role `json:"role"`
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))},
		UserID:           "u",
		Role:             auth.RoleUser,
	})
	s, _ := tok.SignedString([]byte(testSecret))
	v := auth.NewJWTValidator(testSecret)
	_, err := v.Validate(s)
	require.Error(t, err)
}

// --- APIKeyValidator ---

func TestAPIKeyValidator_valid(t *testing.T) {
	v := auth.NewAPIKeyValidator(map[string]string{"key-abc": "PartnerX"})
	identity, err := v.Validate("key-abc")
	require.NoError(t, err)
	assert.Equal(t, "key-abc", identity.VendorID)
	assert.Equal(t, "PartnerX", identity.Name)
}

func TestAPIKeyValidator_invalid(t *testing.T) {
	v := auth.NewAPIKeyValidator(map[string]string{"key-abc": "PartnerX"})
	_, err := v.Validate("wrong-key")
	require.Error(t, err)
	code, _ := errs.Parse(err)
	assert.Equal(t, "INVALID_API_KEY", code)
}

func TestAPIKeyValidator_empty(t *testing.T) {
	v := auth.NewAPIKeyValidator(map[string]string{})
	_, err := v.Validate("")
	require.Error(t, err)
	code, _ := errs.Parse(err)
	assert.Equal(t, "MISSING_API_KEY", code)
}

// --- Context helpers ---

func TestClaimsRoundtrip(t *testing.T) {
	c := &auth.Claims{UserID: "u1", Role: auth.RoleAdmin}
	ctx := auth.WithClaims(context.Background(), c)
	got, ok := auth.ClaimsFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, c, got)
}

func TestVendorRoundtrip(t *testing.T) {
	v := &auth.VendorIdentity{VendorID: "k", Name: "Corp"}
	ctx := auth.WithVendorIdentity(context.Background(), v)
	got, ok := auth.VendorIdentityFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, v, got)
}

func TestClaimsFromContext_missing(t *testing.T) {
	_, ok := auth.ClaimsFromContext(context.Background())
	assert.False(t, ok)
}

// --- RequireAdmin / RequireSuperAdmin ---

func TestRequireAdmin(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Role: auth.RoleAdmin})
	assert.NoError(t, auth.RequireAdmin(ctx))

	ctx2 := auth.WithClaims(context.Background(), &auth.Claims{Role: auth.RoleSuperAdmin})
	assert.NoError(t, auth.RequireAdmin(ctx2))

	ctx3 := auth.WithClaims(context.Background(), &auth.Claims{Role: auth.RoleUser})
	assert.Error(t, auth.RequireAdmin(ctx3))
}

func TestRequireSuperAdmin(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Role: auth.RoleSuperAdmin})
	assert.NoError(t, auth.RequireSuperAdmin(ctx))

	ctx2 := auth.WithClaims(context.Background(), &auth.Claims{Role: auth.RoleAdmin})
	assert.Error(t, auth.RequireSuperAdmin(ctx2))
}

// --- NewAuthInterceptor ---

func TestNewAuthInterceptor_public(t *testing.T) {
	jv := auth.NewJWTValidator(testSecret)
	av := auth.NewAPIKeyValidator(nil)
	policy := func(string) auth.AuthMode { return auth.AuthModePublic }
	interceptor := auth.NewAuthInterceptor(jv, av, policy)

	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	called := false
	_, err := interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		called = true
		return nil, nil
	})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestNewAuthInterceptor_jwt(t *testing.T) {
	jv := auth.NewJWTValidator(testSecret)
	av := auth.NewAPIKeyValidator(nil)
	policy := func(string) auth.AuthMode { return auth.AuthModeJWT }
	interceptor := auth.NewAuthInterceptor(jv, av, policy)

	tok := makeToken(t, "user-1", auth.RoleUser, testSecret)
	ctx := ctxWithAuth(tok)
	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}

	var gotClaims *auth.Claims
	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		gotClaims, _ = auth.ClaimsFromContext(ctx)
		return nil, nil
	})
	require.NoError(t, err)
	require.NotNil(t, gotClaims)
	assert.Equal(t, "user-1", gotClaims.UserID)
}

func TestNewAuthInterceptor_apiKey(t *testing.T) {
	jv := auth.NewJWTValidator(testSecret)
	av := auth.NewAPIKeyValidator(map[string]string{"k": "Corp"})
	policy := func(string) auth.AuthMode { return auth.AuthModeAPIKey }
	interceptor := auth.NewAuthInterceptor(jv, av, policy)

	ctx := ctxWithAPIKey("k")
	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	var gotVendor *auth.VendorIdentity
	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		gotVendor, _ = auth.VendorIdentityFromContext(ctx)
		return nil, nil
	})
	require.NoError(t, err)
	require.NotNil(t, gotVendor)
	assert.Equal(t, "Corp", gotVendor.Name)
}

func TestNewAuthInterceptor_missingJWT(t *testing.T) {
	jv := auth.NewJWTValidator(testSecret)
	av := auth.NewAPIKeyValidator(nil)
	policy := func(string) auth.AuthMode { return auth.AuthModeJWT }
	interceptor := auth.NewAuthInterceptor(jv, av, policy)

	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}
	_, err := interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})
	require.Error(t, err)
	code, _ := errs.Parse(err)
	assert.Equal(t, "MISSING_ACCESS_TOKEN", code)
}
