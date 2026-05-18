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

func makeToken(t *testing.T, identityID string, roles []string, secret string) string {
	t.Helper()
	type jwtClaims struct {
		jwt.RegisteredClaims
		IdentityID string   `json:"iid"`
		Scope      string   `json:"scope"`
		Roles      []string `json:"roles"`
		Email      string   `json:"email"`
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		IdentityID:       identityID,
		Roles:            roles,
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
	tok := makeToken(t, "user-1", []string{"user"}, testSecret)
	claims, err := v.Validate(tok)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.IdentityID)
	assert.Contains(t, claims.Roles, "user")
}

func TestJWTValidator_wrongSecret(t *testing.T) {
	v := auth.NewJWTValidator(testSecret)
	tok := makeToken(t, "user-1", []string{"user"}, "wrong-secret")
	_, err := v.Validate(tok)
	require.Error(t, err)
	code, _ := errs.Parse(err)
	assert.Equal(t, "INVALID_TOKEN", code)
}

func TestJWTValidator_expired(t *testing.T) {
	type jwtClaims struct {
		jwt.RegisteredClaims
		IdentityID string   `json:"iid"`
		Roles      []string `json:"roles"`
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))},
		IdentityID:       "u",
		Roles:            []string{"user"},
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
	c := &auth.Claims{IdentityID: "u1", Roles: []string{"admin"}}
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
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Roles: []string{"admin"}})
	assert.NoError(t, auth.RequireAdmin(ctx))

	ctx2 := auth.WithClaims(context.Background(), &auth.Claims{Roles: []string{"operator"}})
	assert.NoError(t, auth.RequireAdmin(ctx2))

	ctx3 := auth.WithClaims(context.Background(), &auth.Claims{Roles: []string{"user"}})
	assert.Error(t, auth.RequireAdmin(ctx3))
}

func TestRequireSuperAdmin(t *testing.T) {
	ctx := auth.WithClaims(context.Background(), &auth.Claims{Roles: []string{"admin"}})
	assert.NoError(t, auth.RequireSuperAdmin(ctx))

	ctx2 := auth.WithClaims(context.Background(), &auth.Claims{Roles: []string{"operator"}})
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

	tok := makeToken(t, "user-1", []string{"user"}, testSecret)
	ctx := ctxWithAuth(tok)
	info := &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}

	var gotClaims *auth.Claims
	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		gotClaims, _ = auth.ClaimsFromContext(ctx)
		return nil, nil
	})
	require.NoError(t, err)
	require.NotNil(t, gotClaims)
	assert.Equal(t, "user-1", gotClaims.IdentityID)
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

// --- GenerateToken ---

func TestGenerateToken_roundtrip(t *testing.T) {
	claims := &auth.Claims{
		IdentityID: "user-42",
		Roles:      []string{"admin"},
		Email:      "admin@example.com",
	}
	tok, err := auth.GenerateToken(claims, testSecret, time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	v := auth.NewJWTValidator(testSecret)
	got, err := v.Validate(tok)
	require.NoError(t, err)
	assert.Equal(t, "user-42", got.IdentityID)
	assert.Contains(t, got.Roles, "admin")
}

func TestGenerateToken_expired(t *testing.T) {
	claims := &auth.Claims{IdentityID: "user-1", Roles: []string{"user"}}
	tok, err := auth.GenerateToken(claims, testSecret, -time.Minute)
	require.NoError(t, err)

	v := auth.NewJWTValidator(testSecret)
	_, err = v.Validate(tok)
	require.Error(t, err)
}
