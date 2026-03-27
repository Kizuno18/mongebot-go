package netutil

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	attempts := 0
	result, err := Retry(context.Background(), RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}, func(_ context.Context, attempt int) (string, error) {
		attempts++
		if attempt < 2 {
			return "", errors.New("not yet")
		}
		return "success", nil
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_AllFail(t *testing.T) {
	_, err := Retry(context.Background(), RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    5 * time.Millisecond,
	}, func(_ context.Context, _ int) (int, error) {
		return 0, errors.New("fail")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errors.New("fail")) {
		// Just check it wraps the message
		if err.Error() != "after 2 attempts: fail" {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := Retry(ctx, RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Second,
	}, func(_ context.Context, _ int) (int, error) {
		return 0, errors.New("fail")
	})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRandomDuration(t *testing.T) {
	min := 5 * time.Second
	max := 10 * time.Second

	for range 100 {
		d := RandomDuration(min, max)
		if d < min || d > max {
			t.Errorf("duration %v out of range [%v, %v]", d, min, max)
		}
	}
}

func TestRandomDuration_MinEqualsMax(t *testing.T) {
	d := RandomDuration(5*time.Second, 5*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
}

func TestIsTemporaryError(t *testing.T) {
	temporary := []int{408, 429, 500, 502, 503, 504}
	for _, code := range temporary {
		if !IsTemporaryError(code) {
			t.Errorf("expected %d to be temporary", code)
		}
	}

	nonTemporary := []int{200, 301, 400, 401, 403, 404}
	for _, code := range nonTemporary {
		if IsTemporaryError(code) {
			t.Errorf("expected %d to NOT be temporary", code)
		}
	}
}
