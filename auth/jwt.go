package auth

import (
	"github.com/golang-jwt/jwt/v5"

	"github.com/vchitai/grpcx/errs"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Role   Role   `json:"role"`
}

// JWTValidator validates JWT tokens and extracts Claims.
type JWTValidator struct {
	secret []byte
}

// NewJWTValidator creates a JWTValidator using the given HMAC-SHA256 secret.
func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{secret: []byte(secret)}
}

// Validate parses and validates a JWT token string, returning Claims on success.
func (v *JWTValidator) Validate(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errs.Unauthenticated("INVALID_TOKEN", "unexpected signing method")
		}
		return v.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errs.Unauthenticated("INVALID_TOKEN", "invalid or expired token")
	}
	c, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, errs.Unauthenticated("INVALID_TOKEN", "malformed token claims")
	}
	return &Claims{UserID: c.UserID, Role: c.Role}, nil
}
