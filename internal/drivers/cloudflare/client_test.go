package cloudflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amigoer/bucky/internal/core"
)

func newTestClient(t *testing.T, base string) *Client {
	t.Helper()
	client, err := New(Config{
		APIBase:     base,
		OAuthBase:   base,
		ClientID:    "test-client",
		RedirectURI: "http://127.0.0.1:8976/oauth/callback",
		UserAgent:   "Bucky/test",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func TestNewValidatesBaseURLs(t *testing.T) {
	cases := []struct {
		name string
		base string
		ok   bool
	}{
		{"https", "https://api.cloudflare.com/client/v4", true},
		{"loopback http", "http://127.0.0.1:9000", true},
		{"plain http", "http://api.cloudflare.com", false},
		{"credentials", "https://user:pass@api.cloudflare.com", false},
		{"query", "https://api.cloudflare.com/client/v4?x=1", false},
		{"empty host", "https:///client/v4", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := New(Config{APIBase: c.base, RedirectURI: "http://127.0.0.1:8976/oauth/callback"})
			if c.ok && err != nil {
				t.Fatalf("expected %q to be accepted, got %v", c.base, err)
			}
			if !c.ok && err == nil {
				t.Fatalf("expected %q to be rejected", c.base)
			}
		})
	}
}

func TestNewValidatesRedirectURI(t *testing.T) {
	for _, value := range []string{
		"https://127.0.0.1:8976/oauth/callback",
		"http://example.com:8976/oauth/callback",
		"http://127.0.0.1/oauth/callback",
	} {
		if _, err := New(Config{RedirectURI: value}); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	if _, err := New(Config{RedirectURI: "http://127.0.0.1:8976/oauth/callback"}); err != nil {
		t.Fatalf("loopback redirect rejected: %v", err)
	}
}

func TestAccountsSendsBearerToken(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		if r.URL.Path != "/accounts" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"success":true,"result":[{"id":"abc","name":"Amigoer Account"}]}`)
	}))
	defer server.Close()

	accounts, err := newTestClient(t, server.URL).Accounts(context.Background(), "access-token")
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if authorization != "Bearer access-token" {
		t.Fatalf("unexpected authorization header %q", authorization)
	}
	if len(accounts) != 1 || accounts[0].Name != "Amigoer Account" {
		t.Fatalf("unexpected accounts %+v", accounts)
	}
}

func TestStatusMapping(t *testing.T) {
	cases := []struct {
		status    int
		code      string
		retryable bool
	}{
		{400, "invalid_argument", false},
		{401, "credentials_unavailable", false},
		{403, "permission_denied", false},
		{404, "not_found", false},
		{409, "conflict", false},
		{429, "rate_limited", true},
		{503, "unavailable", true},
	}
	for _, c := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(c.status)
			_, _ = io.WriteString(w, `{"success":false,"errors":[{"code":1000,"message":"denied by policy"}]}`)
		}))
		_, err := newTestClient(t, server.URL).Accounts(context.Background(), "access-token")
		server.Close()
		result := core.AsError(err)
		if result == nil || result.Code != c.code || result.Retryable != c.retryable {
			t.Fatalf("status %d mapped to %+v, want %s", c.status, result, c.code)
		}
		if c.status == 403 && result.Message == "" {
			t.Fatalf("expected the provider message to be kept")
		}
	}
}

func TestInvalidEnvelopeIsReported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "<html>not json</html>")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).Accounts(context.Background(), "access-token")
	if result := core.AsError(err); result == nil || result.Code != "provider_error" {
		t.Fatalf("unexpected error %+v", result)
	}
}

func TestCreateTokenBody(t *testing.T) {
	var body map[string]any
	var jurisdiction string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/accounts/acct/tokens/permission_groups":
			if scope := r.URL.Query().Get("scope"); scope != bucketResourceScope {
				t.Errorf("unexpected scope %q", scope)
			}
			_, _ = io.WriteString(w, `{"success":true,"result":[
				{"id":"write-id","name":"Workers R2 Storage Bucket Item Write"},
				{"id":"read-id","name":"Workers R2 Storage Bucket Item Read"}]}`)
		case "/accounts/acct/tokens":
			jurisdiction = r.Header.Get(jurisdictionHeader)
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
			}
			_, _ = io.WriteString(w, `{"success":true,"result":{"id":"token-id","name":"Bucky","value":"token-value"}}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	token, err := newTestClient(t, server.URL).CreateR2Token(context.Background(), "access-token", TokenSpec{
		Name:         "Bucky",
		AccountID:    "acct",
		Bucket:       "media",
		Jurisdiction: "eu",
	})
	if err != nil {
		t.Fatalf("CreateR2Token: %v", err)
	}
	if token.ID != "token-id" || token.Value != "token-value" {
		t.Fatalf("unexpected token %+v", token)
	}
	policies := body["policies"].([]any)
	policy := policies[0].(map[string]any)
	resources := policy["resources"].(map[string]any)
	if _, ok := resources["com.cloudflare.edge.r2.bucket.acct_eu_media"]; !ok {
		t.Fatalf("unexpected resources %+v", resources)
	}
	groups := policy["permission_groups"].([]any)
	if groups[0].(map[string]any)["id"] != "write-id" {
		t.Fatalf("unexpected permission groups %+v", groups)
	}
	if jurisdiction != "" {
		t.Fatalf("the jurisdiction header belongs on bucket listing, not token creation")
	}
}

func TestCreateTokenUsesReadGroup(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/accounts/acct/tokens/permission_groups" {
			_, _ = io.WriteString(w, `{"success":true,"result":[{"id":"read-id","name":"Workers R2 Storage Bucket Item Read"}]}`)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"success":true,"result":{"id":"token-id","value":"token-value"}}`)
	}))
	defer server.Close()

	if _, err := newTestClient(t, server.URL).CreateR2Token(context.Background(), "access-token", TokenSpec{
		AccountID: "acct",
		Bucket:    "media",
		ReadOnly:  true,
	}); err != nil {
		t.Fatalf("CreateR2Token: %v", err)
	}
	policies := body["policies"].([]any)
	policy := policies[0].(map[string]any)
	groups := policy["permission_groups"].([]any)
	if groups[0].(map[string]any)["id"] != "read-id" {
		t.Fatalf("expected the read permission group, got %+v", groups)
	}
}

func TestCreateTokenRejectsUnusableBucketName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).CreateR2Token(context.Background(), "access-token", TokenSpec{
		AccountID: "acct",
		Bucket:    "media_bucket",
	})
	if result := core.AsError(err); result == nil || result.Code != "invalid_argument" {
		t.Fatalf("unexpected error %+v", result)
	}
}

func TestS3CredentialsDerivation(t *testing.T) {
	creds := S3Credentials(Token{ID: "token-id", Value: "token-value"})
	if creds.AccessKeyID != "token-id" {
		t.Fatalf("unexpected access key %q", creds.AccessKeyID)
	}
	const want = "e6c02a5742ea9d4de588eb9b9de7bed43dc17011552186bed3e98b2c5958ff4a"
	if creds.SecretAccessKey != want {
		t.Fatalf("secret = %q, want the SHA-256 hex of the token value", creds.SecretAccessKey)
	}
}

func TestBucketsSendJurisdictionHeader(t *testing.T) {
	var jurisdiction string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jurisdiction = r.Header.Get(jurisdictionHeader)
		_, _ = io.WriteString(w, `{"success":true,"result":{"buckets":[{"name":"media","creation_date":"2026-01-01"}]}}`)
	}))
	defer server.Close()

	buckets, err := newTestClient(t, server.URL).Buckets(context.Background(), "access-token", "acct", "eu")
	if err != nil {
		t.Fatalf("Buckets: %v", err)
	}
	if jurisdiction != "eu" {
		t.Fatalf("unexpected jurisdiction header %q", jurisdiction)
	}
	if len(buckets) != 1 || buckets[0].Name != "media" {
		t.Fatalf("unexpected buckets %+v", buckets)
	}
}
