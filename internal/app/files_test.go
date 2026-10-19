package app

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalTarget(t *testing.T) {
	cases := []struct {
		key, base, want string
		ok              bool
	}{
		{"photos/2024/a.jpg", "photos/", filepath.Join("2024", "a.jpg"), true},
		{"photos/a.jpg", "", "a.jpg", true},
		{"photos/2024/", "photos/", "2024", true},
		{"other/a.jpg", "photos/", "a.jpg", true},
		{"photos/../../etc/passwd", "photos/", "", false},
		{"photos/..", "photos/", "", false},
		{"photos/", "photos/", "photos", true},
		{"/abs", "", "abs", true},
		{"photos//a.jpg", "photos/", "", false},
		{"photos/a\\b.jpg", "photos/", "", false},
	}
	for _, c := range cases {
		got, ok := localTarget(c.key, c.base)
		if ok != c.ok || got != c.want {
			t.Errorf("localTarget(%q, %q) = %q, %v; want %q, %v", c.key, c.base, got, ok, c.want, c.ok)
		}
	}
}

func TestStagedNameKeepsExtension(t *testing.T) {
	a, b := stagedName("images/2026/x.WEBP"), stagedName("images/2026/x.WEBP")
	if a == b || !strings.HasSuffix(a, ".webp") || strings.ContainsAny(a, "/\\") {
		t.Fatalf("names = %q, %q", a, b)
	}
	if name := stagedName("noext"); strings.Contains(name, ".") {
		t.Fatalf("name = %q", name)
	}
}
