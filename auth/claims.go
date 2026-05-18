package auth

// Role represents a user's authorization level — kept for backward-compatible
// helpers such as toProtoRole in consumer apps.
type Role string

const (
	RoleUser       Role = "user"
	RoleAgent      Role = "agent"
	RoleOperator   Role = "operator"
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

// Claims are the JWT payload injected into context after successful authentication.
type Claims struct {
	IdentityID string   `json:"iid"`
	Scope      string   `json:"scope"` // "consumer" | "admin" | "operator"
	Roles      []string `json:"roles"`
	Email      string   `json:"email"`
}

// VendorIdentity is injected into context after successful API key authentication.
type VendorIdentity struct {
	VendorID string
	Name     string
}
