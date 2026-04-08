package api

import (
	"regexp"
	"strings"
)

// uuidRegex strictly matches standard UUID v4 format.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isValidID checks if the provided string is a safe, valid UUID.
// This prevents path traversal attacks in notebook operations.
func isValidID(id string) bool {
	return uuidRegex.MatchString(id)
}

// sanitizeError removes the API key from any error message string to prevent leaks.
func sanitizeError(err error, apiKey string) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if apiKey != "" && len(apiKey) > 4 {
		msg = strings.ReplaceAll(msg, apiKey, "REDACTED_API_KEY")
	}
	return msg
}
