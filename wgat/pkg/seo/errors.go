package seo

import (
	"errors"
	"fmt"
)

// Error codes for SEO operations
const (
	ErrCodeInvalidURL     = "INVALID_URL"
	ErrCodeFetchFailed    = "FETCH_FAILED"
	ErrCodeParseFailed    = "PARSE_FAILED"
	ErrCodeTimeout        = "TIMEOUT"
	ErrCodeRateLimited    = "RATE_LIMITED"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeInternal       = "INTERNAL_ERROR"
)

// SEOError represents a custom error with code
type SEOError struct {
	Code    string
	Message string
	Cause   error
}

func (e *SEOError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *SEOError) Unwrap() error {
	return e.Cause
}

// NewSEOError creates a new SEO error
func NewSEOError(code, message string, cause error) *SEOError {
	return &SEOError{Code: code, Message: message, Cause: cause}
}

// Is checks if the error matches a code
func (e *SEOError) Is(target error) bool {
	se, ok := target.(*SEOError)
	if !ok {
		return false
	}
	return se.Code == e.Code
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// AggregateError collects multiple errors
type AggregateError struct {
	Errors []error
}

func (e *AggregateError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	msg := "multiple errors:"
	for _, err := range e.Errors {
		msg += fmt.Sprintf("\n  - %v", err)
	}
	return msg
}

func (e *AggregateError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

func (e *AggregateError) HasErrors() bool {
	return len(e.Errors) > 0
}

// IsTemporary checks if an error is temporary (retryable)
func IsTemporary(err error) bool {
	if err == nil {
		return false
	}
	var seoErr *SEOError
	if errors.As(err, &seoErr) {
		switch seoErr.Code {
		case ErrCodeTimeout, ErrCodeRateLimited, ErrCodeFetchFailed:
			return true
		}
	}
	return false
}