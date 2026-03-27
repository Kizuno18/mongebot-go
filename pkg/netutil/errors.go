// Package netutil - categorized error types with retry policies.
// Provides structured error classification for network, auth, rate-limit, and platform errors.
package netutil

import (
	"errors"
	"fmt"
	"time"
)

// ErrorCategory classifies errors for automated handling.
type ErrorCategory int

const (
	ErrCategoryUnknown    ErrorCategory = iota
	ErrCategoryNetwork                         // Connection timeout, DNS, TLS errors
	ErrCategoryAuth                            // 401, token expired, invalid credentials
	ErrCategoryRateLimit                       // 429, too many requests
	ErrCategoryPlatform                        // Platform-specific API errors
	ErrCategoryProxy                           // Proxy connection/auth failures
	ErrCategoryStream                          // Stream offline, HLS errors
	ErrCategoryInternal                        // Internal logic errors
)

// String returns a human-readable error category.
func (c ErrorCategory) String() string {
	names := [...]string{"unknown", "network", "auth", "rate_limit", "platform", "proxy", "stream", "internal"}
	if int(c) < len(names) {
		return names[c]
	}
	return "unknown"
}

// ShouldRetry returns whether errors in this category are retryable.
func (c ErrorCategory) ShouldRetry() bool {
	switch c {
	case ErrCategoryNetwork, ErrCategoryRateLimit, ErrCategoryProxy:
		return true
	default:
		return false
	}
}

// RetryDelay returns the recommended delay before retrying for this category.
func (c ErrorCategory) RetryDelay() time.Duration {
	switch c {
	case ErrCategoryNetwork:
		return 5 * time.Second
	case ErrCategoryRateLimit:
		return 30 * time.Second
	case ErrCategoryProxy:
		return 2 * time.Second
	default:
		return 10 * time.Second
	}
}

// CategorizedError wraps an error with a category and context.
type CategorizedError struct {
	Category ErrorCategory
	Message  string
	Cause    error
	Context  map[string]string // Additional context (component, worker, etc.)
}

// Error implements the error interface.
func (e *CategorizedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Category, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/errors.As.
func (e *CategorizedError) Unwrap() error {
	return e.Cause
}

// NewError creates a categorized error.
func NewError(category ErrorCategory, message string, cause error) *CategorizedError {
	return &CategorizedError{
		Category: category,
		Message:  message,
		Cause:    cause,
	}
}

// WithContext adds context key-value pairs to the error.
func (e *CategorizedError) WithContext(kv ...string) *CategorizedError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	for i := 0; i+1 < len(kv); i += 2 {
		e.Context[kv[i]] = kv[i+1]
	}
	return e
}

// Convenience constructors

// NetworkError creates a network-category error.
func NetworkError(msg string, cause error) *CategorizedError {
	return NewError(ErrCategoryNetwork, msg, cause)
}

// AuthError creates an auth-category error.
func AuthError(msg string, cause error) *CategorizedError {
	return NewError(ErrCategoryAuth, msg, cause)
}

// RateLimitError creates a rate-limit-category error.
func RateLimitError(msg string, cause error) *CategorizedError {
	return NewError(ErrCategoryRateLimit, msg, cause)
}

// PlatformError creates a platform-category error.
func PlatformError(msg string, cause error) *CategorizedError {
	return NewError(ErrCategoryPlatform, msg, cause)
}

// ProxyError creates a proxy-category error.
func ProxyError(msg string, cause error) *CategorizedError {
	return NewError(ErrCategoryProxy, msg, cause)
}

// StreamError creates a stream-category error.
func StreamError(msg string, cause error) *CategorizedError {
	return NewError(ErrCategoryStream, msg, cause)
}

// CategorizeHTTPError maps an HTTP status code to an error category.
func CategorizeHTTPError(statusCode int, msg string) *CategorizedError {
	switch {
	case statusCode == 401 || statusCode == 403:
		return AuthError(msg, nil)
	case statusCode == 429:
		return RateLimitError(msg, nil)
	case statusCode >= 500:
		return PlatformError(msg, nil)
	case statusCode == 407:
		return ProxyError(msg, nil)
	default:
		return NewError(ErrCategoryUnknown, msg, nil)
	}
}

// GetCategory extracts the error category from any error.
// Returns ErrCategoryUnknown if the error is not a CategorizedError.
func GetCategory(err error) ErrorCategory {
	var catErr *CategorizedError
	if errors.As(err, &catErr) {
		return catErr.Category
	}
	return ErrCategoryUnknown
}

// IsRetryable checks if an error should be retried.
func IsRetryable(err error) bool {
	return GetCategory(err).ShouldRetry()
}
