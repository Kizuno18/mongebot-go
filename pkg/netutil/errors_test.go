package netutil

import (
	"errors"
	"testing"
	"time"
)

func TestErrorCategory_String(t *testing.T) {
	tests := []struct {
		cat  ErrorCategory
		want string
	}{
		{ErrCategoryNetwork, "network"},
		{ErrCategoryAuth, "auth"},
		{ErrCategoryRateLimit, "rate_limit"},
		{ErrCategoryProxy, "proxy"},
		{ErrCategoryStream, "stream"},
	}
	for _, tt := range tests {
		if got := tt.cat.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.cat, got, tt.want)
		}
	}
}

func TestErrorCategory_ShouldRetry(t *testing.T) {
	retryable := []ErrorCategory{ErrCategoryNetwork, ErrCategoryRateLimit, ErrCategoryProxy}
	for _, c := range retryable {
		if !c.ShouldRetry() {
			t.Errorf("%s should be retryable", c)
		}
	}

	nonRetryable := []ErrorCategory{ErrCategoryAuth, ErrCategoryPlatform, ErrCategoryStream, ErrCategoryInternal}
	for _, c := range nonRetryable {
		if c.ShouldRetry() {
			t.Errorf("%s should NOT be retryable", c)
		}
	}
}

func TestErrorCategory_RetryDelay(t *testing.T) {
	if ErrCategoryRateLimit.RetryDelay() < 20*time.Second {
		t.Error("rate limit delay should be >= 20s")
	}
	if ErrCategoryNetwork.RetryDelay() < 3*time.Second {
		t.Error("network delay should be >= 3s")
	}
}

func TestCategorizedError_Error(t *testing.T) {
	err := NetworkError("connection failed", errors.New("timeout"))
	expected := "[network] connection failed: timeout"
	if err.Error() != expected {
		t.Errorf("got %q, want %q", err.Error(), expected)
	}
}

func TestCategorizedError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := AuthError("token expired", cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the root cause")
	}
}

func TestCategorizedError_WithContext(t *testing.T) {
	err := ProxyError("connection refused", nil).
		WithContext("proxy", "1.2.3.4:8080", "worker", "v-abc123")

	if err.Context["proxy"] != "1.2.3.4:8080" {
		t.Errorf("expected proxy context, got %v", err.Context)
	}
	if err.Context["worker"] != "v-abc123" {
		t.Errorf("expected worker context, got %v", err.Context)
	}
}

func TestCategorizeHTTPError(t *testing.T) {
	tests := []struct {
		code int
		cat  ErrorCategory
	}{
		{401, ErrCategoryAuth},
		{403, ErrCategoryAuth},
		{429, ErrCategoryRateLimit},
		{500, ErrCategoryPlatform},
		{502, ErrCategoryPlatform},
		{407, ErrCategoryProxy},
		{404, ErrCategoryUnknown},
	}

	for _, tt := range tests {
		err := CategorizeHTTPError(tt.code, "test")
		if err.Category != tt.cat {
			t.Errorf("HTTP %d: expected %s, got %s", tt.code, tt.cat, err.Category)
		}
	}
}

func TestGetCategory(t *testing.T) {
	err := RateLimitError("too many requests", nil)
	if GetCategory(err) != ErrCategoryRateLimit {
		t.Error("expected rate_limit category")
	}

	plainErr := errors.New("plain error")
	if GetCategory(plainErr) != ErrCategoryUnknown {
		t.Error("expected unknown category for plain error")
	}
}

func TestIsRetryable(t *testing.T) {
	if !IsRetryable(NetworkError("fail", nil)) {
		t.Error("network errors should be retryable")
	}
	if IsRetryable(AuthError("expired", nil)) {
		t.Error("auth errors should NOT be retryable")
	}
}
