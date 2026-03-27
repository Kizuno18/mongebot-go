// Package token - multi-format token import including browser cookie extraction.
package token

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ImportFormat identifies the source format of imported tokens.
type ImportFormat int

const (
	FormatRawText ImportFormat = iota // One token per line
	FormatJSON                       // JSON array of strings
	FormatNetscape                   // Netscape cookie file format
	FormatEditThisCookie             // EditThisCookie JSON export
)

// ImportResult holds the outcome of a token import operation.
type ImportResult struct {
	Added    int      `json:"added"`
	Skipped  int      `json:"skipped"`
	Invalid  int      `json:"invalid"`
	Errors   []string `json:"errors,omitempty"`
}

// ImportFromFile reads tokens from a file, auto-detecting the format.
func ImportFromFile(path string) ([]string, ImportFormat, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, FormatRawText, err
	}
	return ImportFromBytes(data)
}

// ImportFromBytes parses tokens from raw bytes, auto-detecting the format.
func ImportFromBytes(data []byte) ([]string, ImportFormat, error) {
	content := strings.TrimSpace(string(data))

	// Try JSON array first
	if strings.HasPrefix(content, "[") {
		tokens, err := parseJSONArray(content)
		if err == nil && len(tokens) > 0 {
			return tokens, FormatJSON, nil
		}

		// Try EditThisCookie format
		tokens, err = parseEditThisCookie(content)
		if err == nil && len(tokens) > 0 {
			return tokens, FormatEditThisCookie, nil
		}
	}

	// Try Netscape cookie file
	if strings.Contains(content, ".twitch.tv") && strings.Contains(content, "auth-token") {
		tokens := parseNetscapeCookies(content)
		if len(tokens) > 0 {
			return tokens, FormatNetscape, nil
		}
	}

	// Default: raw text (one per line)
	tokens := parseRawText(content)
	return tokens, FormatRawText, nil
}

// parseRawText parses one token per line, skipping empty lines and comments.
func parseRawText(content string) []string {
	var tokens []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		// Basic validation: tokens should be alphanumeric, min 20 chars
		if len(line) >= 20 && isAlphanumeric(line) {
			tokens = append(tokens, line)
		}
	}
	return tokens
}

// parseJSONArray parses a JSON array of token strings.
func parseJSONArray(content string) ([]string, error) {
	var arr []string
	if err := json.Unmarshal([]byte(content), &arr); err != nil {
		return nil, err
	}
	var valid []string
	for _, t := range arr {
		t = strings.TrimSpace(t)
		if len(t) >= 20 {
			valid = append(valid, t)
		}
	}
	return valid, nil
}

// parseEditThisCookie extracts auth-token from EditThisCookie JSON export.
func parseEditThisCookie(content string) ([]string, error) {
	var cookies []struct {
		Name   string `json:"name"`
		Value  string `json:"value"`
		Domain string `json:"domain"`
	}

	if err := json.Unmarshal([]byte(content), &cookies); err != nil {
		return nil, err
	}

	var tokens []string
	for _, cookie := range cookies {
		if cookie.Name == "auth-token" && strings.Contains(cookie.Domain, "twitch") {
			if len(cookie.Value) >= 20 {
				tokens = append(tokens, cookie.Value)
			}
		}
	}
	return tokens, nil
}

// parseNetscapeCookies extracts auth-token from Netscape cookie file format.
// Format: domain\tpath\tsecure\texpiry\tname\tvalue
func parseNetscapeCookies(content string) []string {
	var tokens []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) >= 7 {
			name := fields[5]
			value := fields[6]
			domain := fields[0]

			if name == "auth-token" && strings.Contains(domain, "twitch") && len(value) >= 20 {
				tokens = append(tokens, value)
			}
		}
	}
	return tokens
}

// isAlphanumeric checks if a string contains only alphanumeric characters.
func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// FormatName returns a human-readable name for the import format.
func (f ImportFormat) String() string {
	names := [...]string{"Raw Text", "JSON Array", "Netscape Cookies", "EditThisCookie"}
	if int(f) < len(names) {
		return names[f]
	}
	return fmt.Sprintf("Unknown(%d)", f)
}
