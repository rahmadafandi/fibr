// Copyright 2026 Rahmad Afandi. MIT License.

package randhex

import (
	"regexp"
	"testing"
)

func TestString(t *testing.T) {
	a, err := String(16)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(a) {
		t.Fatalf("String(16) = %q, want 32 hex characters", a)
	}
	b, _ := String(16)
	if a == b {
		t.Fatal("two calls returned the same value")
	}
	if s, _ := String(0); s != "" {
		t.Fatalf("String(0) = %q, want empty", s)
	}
}
