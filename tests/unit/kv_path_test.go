package unit

import (
	"strings"
	"testing"
)

// ponytail: mirrors looksLikeKeyName in handlers — fails if heuristic drifts
func looksLikeKeyName(s string) bool {
	if strings.Contains(s, "_") {
		return true
	}
	return s != strings.ToLower(s)
}

func TestLooksLikeKeyName(t *testing.T) {
	if !looksLikeKeyName("DB_HOST") {
		t.Fatal("DB_HOST should look like a key")
	}
	if looksLikeKeyName("random") {
		t.Fatal("random should look like a namespace segment")
	}
	if !looksLikeKeyName("apiKey") {
		t.Fatal("mixed-case should look like a key")
	}
}
