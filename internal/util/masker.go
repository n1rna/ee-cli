package util

import (
	"path/filepath"
	"strings"
)

// SensitivePatterns contains patterns for environment variables that likely contain sensitive data
var SensitivePatterns = []string{
	"*PASSWORD*",
	"*SECRET*",
	"*KEY*",
	"*TOKEN*",
	"*API_KEY*",
	"*ACCESS_KEY*",
	"*PRIVATE_KEY*",
	"*PUBLIC_KEY*",
	"*ENCRYPTION_KEY*",
	"*AUTH*",
	"*CREDENTIAL*",
	"*CERT*",
	"*SSL*",
	"*TLS*",
	"*OAUTH*",
	"*JWT*",
	"*BEARER*",
	"*PASSPHRASE*",
	"*PIN*",
	"*SEED*",
	"*HASH*",
	"*SIGNATURE*",
	"*WEBHOOK*",
	"*DATABASE_URL*",
	"*DB_PASSWORD*",
	"*REDIS_PASSWORD*",
	"*SMTP_PASSWORD*",
	"*FTP_PASSWORD*",
}

// MaskSensitiveValue checks if an environment variable name matches any sensitive pattern
// and returns a masked value if it does, otherwise returns the original value
func MaskSensitiveValue(key, value string) string {
	if IsSensitive(key) {
		return MaskValue(value)
	}
	return value
}

// IsSensitive checks if an environment variable name matches any sensitive pattern
func IsSensitive(key string) bool {
	upperKey := strings.ToUpper(key)

	for _, pattern := range SensitivePatterns {
		matched, err := filepath.Match(pattern, upperKey)
		if err != nil {
			continue // Skip invalid patterns
		}
		if matched {
			return true
		}
	}
	return false
}

// MaskValue creates a masked version of a value
// Shows first 2 and last 2 characters for values longer than 8 characters
// Shows asterisks for shorter values
func MaskValue(value string) string {
	if len(value) == 0 {
		return value
	}

	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}

	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}

	// For longer values, show first 2 and last 2 characters
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
