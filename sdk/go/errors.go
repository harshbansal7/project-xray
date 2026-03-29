package xray

import "fmt"

// ValidationError indicates instrumentation input is invalid.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// APIError captures non-2xx API responses.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("xray API request failed with status %d: %s", e.StatusCode, e.Body)
}
