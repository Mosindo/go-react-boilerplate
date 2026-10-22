package platformerrors

import "fmt"

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// ValidationError describes invalid client input. Handlers map it to HTTP 400.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// Invalid builds a ValidationError.
func Invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
