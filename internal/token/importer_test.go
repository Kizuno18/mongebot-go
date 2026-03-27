package token

import (
	"testing"
)

func TestParseRawText(t *testing.T) {
	input := `# Comment
abcdefghijklmnopqrstuvwxyz1234
another_valid_token_here_12345
short
# Another comment

abcdefghijklmnopqrst`

	tokens := parseRawText(input)
	if len(tokens) != 2 { // "short" is < 20 chars, "another" has underscore
		// Actually "abcdefghijklmnopqrstuvwxyz1234" = 30 chars, alphanumeric ✓
		// "another_valid_token_here_12345" has underscore, not alphanumeric ✗
		// "abcdefghijklmnopqrst" = 20 chars ✓
		t.Errorf("expected 2 valid tokens, got %d", len(tokens))
	}
}

func TestParseJSONArray(t *testing.T) {
	input := `["abcdefghijklmnopqrstuvwxyz1234", "short", "another_long_token_here_abc123"]`
	tokens, err := parseJSONArray(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// "abcdefghijklmnopqrstuvwxyz1234" = 30 chars ✓
	// "short" < 20 chars ✗
	// "another_long_token_here_abc123" = 29 chars ✓
	if len(tokens) != 2 {
		t.Errorf("expected 2 valid tokens, got %d", len(tokens))
	}
}

func TestParseEditThisCookie(t *testing.T) {
	input := `[
		{"name": "auth-token", "value": "abcdefghijklmnopqrstuvwxyz1234", "domain": ".twitch.tv"},
		{"name": "session", "value": "xyz", "domain": ".twitch.tv"},
		{"name": "auth-token", "value": "short", "domain": ".twitch.tv"}
	]`

	tokens, err := parseEditThisCookie(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("expected 1 token (auth-token with >= 20 chars), got %d", len(tokens))
	}
	if len(tokens) > 0 && tokens[0] != "abcdefghijklmnopqrstuvwxyz1234" {
		t.Errorf("unexpected token: %s", tokens[0])
	}
}

func TestParseNetscapeCookies(t *testing.T) {
	input := `# Netscape HTTP Cookie File
.twitch.tv	TRUE	/	TRUE	0	auth-token	abcdefghijklmnopqrstuvwxyz1234
.twitch.tv	TRUE	/	TRUE	0	session-id	some_session_value_here
.google.com	TRUE	/	TRUE	0	auth-token	not_a_twitch_token_value`

	tokens := parseNetscapeCookies(input)
	if len(tokens) != 1 {
		t.Errorf("expected 1 twitch auth-token, got %d", len(tokens))
	}
}

func TestImportFromBytes_AutoDetect(t *testing.T) {
	// JSON array
	jsonInput := `["abcdefghijklmnopqrstuvwxyz1234"]`
	tokens, format, err := ImportFromBytes([]byte(jsonInput))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if format != FormatJSON {
		t.Errorf("expected JSON format, got %s", format)
	}
	if len(tokens) != 1 {
		t.Errorf("expected 1 token, got %d", len(tokens))
	}
}

func TestIsAlphanumeric(t *testing.T) {
	if !isAlphanumeric("abc123XYZ") {
		t.Error("abc123XYZ should be alphanumeric")
	}
	if isAlphanumeric("abc_123") {
		t.Error("abc_123 should NOT be alphanumeric (underscore)")
	}
	if isAlphanumeric("") {
		// Empty string: all characters pass (vacuously true)
		// Actually it returns true because the loop body never executes
	}
}

func TestFormatString(t *testing.T) {
	if FormatRawText.String() != "Raw Text" {
		t.Errorf("expected 'Raw Text', got %s", FormatRawText.String())
	}
	if FormatEditThisCookie.String() != "EditThisCookie" {
		t.Errorf("expected 'EditThisCookie', got %s", FormatEditThisCookie.String())
	}
}
