package transform

import "strings"

// SecretFields lists field names that should be stripped for security
var SecretFields = map[string]bool{
	"token":       true,
	"credential":  true,
	"credentials": true,
	"password":    true,
	"secret":      true,
	"username":    true, // TURN username contains sensitive info
	"ice-pwd":     true,
	"ice-ufrag":   true,
}

// IsSecretField returns true if the field name indicates sensitive data
func IsSecretField(name string) bool {
	lower := strings.ToLower(name)
	return SecretFields[lower]
}

// StripSecrets removes sensitive fields from a map recursively
func StripSecrets(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if IsSecretField(k) {
			continue
		}

		// Recursively handle nested maps
		switch nested := v.(type) {
		case map[string]interface{}:
			v = StripSecrets(nested)
		case []interface{}:
			v = stripSecretsFromSlice(nested)
		}

		result[k] = v
	}
	return result
}

func stripSecretsFromSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		if m, ok := v.(map[string]interface{}); ok {
			result[i] = StripSecrets(m)
		} else {
			result[i] = v
		}
	}
	return result
}
