package auth

import (
	"github.com/vchitai/grpcx/errs"
)

// APIKeyValidator looks up API keys and returns the associated VendorIdentity.
type APIKeyValidator struct {
	keys map[string]string // api_key → vendor_name
}

// NewAPIKeyValidator creates a validator from a key→name map.
func NewAPIKeyValidator(keys map[string]string) *APIKeyValidator {
	return &APIKeyValidator{keys: keys}
}

// Validate checks the given API key and returns the associated VendorIdentity.
func (v *APIKeyValidator) Validate(key string) (*VendorIdentity, error) {
	if key == "" {
		return nil, errs.Unauthenticated("MISSING_API_KEY", "missing api key")
	}
	name, ok := v.keys[key]
	if !ok {
		return nil, errs.Unauthenticated("INVALID_API_KEY", "invalid api key")
	}
	return &VendorIdentity{VendorID: key, Name: name}, nil
}
