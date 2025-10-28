package main

import (
	"regexp"
	"strings"
	"testing"
)

// cleanProfanity replaces profane words with ****.
// Words with ANY punctuation attached are considered different words and NOT replaced.
// Case-insensitive matching (Kerfuffle and kerfuffle both get replaced).
func cleanProfanity(s string) string {
	profane := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	// Split on whitespace while preserving it
	re := regexp.MustCompile(`(\S+|\s+)`)
	parts := re.FindAllString(s, -1)

	var result strings.Builder
	for _, part := range parts {
		// If part is purely whitespace, write it unchanged
		if len(strings.TrimSpace(part)) == 0 {
			result.WriteString(part)
			continue
		}

		// Check if this token is purely alphanumeric (no punctuation)
		isPureWord := true
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				isPureWord = false
				break
			}
		}

		// Only replace if it's a pure word (no punctuation) and matches profane list
		if isPureWord && profane[strings.ToLower(part)] {
			result.WriteString("****")
		} else {
			result.WriteString(part)
		}
	}
	return result.String()
}

// === TEST CASES ===

func TestCleanProfanity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic lowercase",
			input:    "this is a kerfuffle",
			expected: "this is a ****",
		},
		{
			name:     "mixed case",
			input:    "Kerfuffle SHARBERT Fornax",
			expected: "**** **** ****",
		},
		{
			name:     "punctuation blocks replacement - exclamation",
			input:    "kerfuffle! sharbert, fornax.",
			expected: "kerfuffle! sharbert, fornax.",
		},
		{
			name:     "mixed case with punctuation blocks",
			input:    "Sharbert! is not sharbert.",
			expected: "Sharbert! is not sharbert.",
		},
		{
			name:     "multiple profane words",
			input:    "A kerfuffle and fornax caused sharbert drama",
			expected: "A **** and **** caused **** drama",
		},
		{
			name:     "leading punctuation blocks",
			input:    "!kerfuffle .sharbert ..fornax",
			expected: "!kerfuffle .sharbert ..fornax",
		},
		{
			name:     "no profanity",
			input:    "just a normal chirp here",
			expected: "just a normal chirp here",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "   \t\n  ",
			expected: "   \t\n  ",
		},
		{
			name:     "partial word match should NOT replace",
			input:    "kerfuffles sharberty fornaxx",
			expected: "kerfuffles sharberty fornaxx",
		},
		{
			name:     "word boundaries matter",
			input:    "kerfuffle fornax sharbert",
			expected: "**** **** ****",
		},
		{
			name:     "profane word at start and end",
			input:    "kerfuffle is a fornax",
			expected: "**** is a ****",
		},
		{
			name:     "numbers in words prevent match",
			input:    "kerfuffle123 sharbert fornax456",
			expected: "kerfuffle123 **** fornax456",
		},
		{
			name:     "apostrophe blocks replacement",
			input:    "it's not kerfuffle's fault",
			expected: "it's not kerfuffle's fault",
		},
		{
			name:     "hyphenated words",
			input:    "kerfuffle-related sharbert-like",
			expected: "kerfuffle-related sharbert-like",
		},
		{
			name:     "example from problem statement",
			input:    "This is a kerfuffle opinion I need to share with the world",
			expected: "This is a **** opinion I need to share with the world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanProfanity(tt.input)
			if got != tt.expected {
				t.Errorf("\nInput:    %q\nGot:      %q\nExpected: %q", tt.input, got, tt.expected)
			}
		})
	}
}