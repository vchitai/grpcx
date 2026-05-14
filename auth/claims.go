package auth

// Role represents a user's authorization level in JWT claims.
type Role string

const (
	RoleUser       Role = "user"
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

// Claims are the JWT payload injected into context after successful authentication.
type Claims struct {
	UserID string `json:"user_id"`
	Role   Role   `json:"role"`
}

// VendorIdentity is injected into context after successful API key authentication.
type VendorIdentity struct {
	VendorID string
	Name     string
}
