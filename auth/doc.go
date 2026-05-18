// Package auth provides authentication primitives for gRPC services.
//
// # Validators
//
// [NewJWTValidator] creates a validator that parses and verifies HMAC-SHA256 JWTs.
// [NewAPIKeyValidator] creates a validator that looks up API keys in a static map.
//
// # Interceptor
//
// [NewAuthInterceptor] returns a unary interceptor that authenticates each request
// according to a caller-supplied [RoutePolicy]. Example wiring:
//
//	policy := func(method string) auth.AuthMode {
//	    if strings.HasPrefix(method, "/api.health.v1.") {
//	        return auth.AuthModePublic
//	    }
//	    return auth.AuthModeJWT
//	}
//	interceptor := auth.NewAuthInterceptor(jwtVal, apiKeyVal, policy)
//
// After successful authentication the verified identity is available in the
// handler context:
//
//	claims, ok := auth.ClaimsFromContext(ctx)         // JWT: IdentityID, Scope, Roles, Email
//	vendor, ok := auth.VendorIdentityFromContext(ctx) // API key: VendorID, Name
package auth
