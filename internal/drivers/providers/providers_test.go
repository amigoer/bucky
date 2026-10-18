package providers

import (
	"strings"
	"testing"

	"github.com/amigoer/bucky/internal/core"
)

func TestR2Endpoint(t *testing.T) {
	account := strings.Repeat("ab", 16)
	cases := []struct {
		jurisdiction string
		want         string
	}{
		{"", "https://" + account + ".r2.cloudflarestorage.com"},
		{"default", "https://" + account + ".r2.cloudflarestorage.com"},
		{"eu", "https://" + account + ".eu.r2.cloudflarestorage.com"},
		{"fedramp", "https://" + account + ".fedramp.r2.cloudflarestorage.com"},
	}
	for _, test := range cases {
		endpoint, err := r2Endpoint(account, test.jurisdiction)
		if err != nil {
			t.Fatalf("Jurisdiction %q was rejected", test.jurisdiction)
		}
		if endpoint != test.want {
			t.Fatalf("Jurisdiction %q produced the wrong endpoint", test.jurisdiction)
		}
	}
	if endpoint, err := r2Endpoint(strings.ToUpper(account), "default"); err != nil || endpoint != "https://"+account+".r2.cloudflarestorage.com" {
		t.Fatal("Account IDs were not normalized")
	}
}

func TestR2EndpointRejectsInvalidConfiguration(t *testing.T) {
	account := strings.Repeat("ab", 16)
	for _, value := range []string{"", "short", strings.Repeat("zz", 16)} {
		if _, err := r2Endpoint(value, "default"); err == nil || core.AsError(err).Code != "invalid_argument" {
			t.Fatalf("Invalid account ID was accepted")
		}
	}
	if _, err := r2Endpoint(account, "unknown"); err == nil || core.AsError(err).Code != "invalid_argument" {
		t.Fatal("Unknown jurisdiction was accepted")
	}
}
