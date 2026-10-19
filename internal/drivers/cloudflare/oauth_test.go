package cloudflare

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/amigoer/bucky/internal/core"
)

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return strconv.Itoa(port)
}

func TestCodeChallengeMatchesRFC7636(t *testing.T) {
	// RFC 7636 appendix B.
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const want = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got := codeChallenge(verifier); got != want {
		t.Fatalf("challenge = %q, want %q", got, want)
	}
}

func TestRandomValueIsUnpaddedBase64URL(t *testing.T) {
	first, err := randomValue(32)
	if err != nil {
		t.Fatalf("randomValue: %v", err)
	}
	second, err := randomValue(32)
	if err != nil {
		t.Fatalf("randomValue: %v", err)
	}
	if first == second || len(first) != 43 || strings.ContainsAny(first, "+/=") {
		t.Fatalf("unexpected verifier %q", first)
	}
}

func TestAuthURLCarriesPKCEAndState(t *testing.T) {
	port := freePort(t)
	client, err := New(Config{
		ClientID:    "client-id",
		Scopes:      []string{"workers-r2.read", "workers-r2.write"},
		RedirectURI: "http://127.0.0.1:" + port + "/oauth/callback",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	flow, err := client.NewFlow()
	if err != nil {
		t.Fatalf("NewFlow: %v", err)
	}
	defer func() { _ = flow.Close() }()

	parsed, err := url.Parse(flow.AuthURL())
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	if !strings.HasSuffix(parsed.Path, "/auth") {
		t.Fatalf("unexpected authorization path %q", parsed.Path)
	}
	query := parsed.Query()
	if query.Get("client_id") != "client-id" || query.Get("response_type") != "code" {
		t.Fatalf("unexpected query %v", query)
	}
	if query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") != codeChallenge(flow.verifier) {
		t.Fatalf("unexpected PKCE parameters %v", query)
	}
	if query.Get("state") == "" || query.Get("redirect_uri") != flow.RedirectURI() {
		t.Fatalf("unexpected state or redirect %v", query)
	}
	if query.Get("scope") != "workers-r2.read workers-r2.write" {
		t.Fatalf("unexpected scopes %q", query.Get("scope"))
	}
}

func TestFlowExchangesTheAuthorizationCode(t *testing.T) {
	port := freePort(t)
	var form url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_ = r.ParseForm()
		form = r.PostForm
		_, _ = io.WriteString(w, `{"access_token":"access","refresh_token":"refresh","token_type":"bearer","expires_in":3600,"scope":"workers-r2.read"}`)
	}))
	defer server.Close()

	client, err := New(Config{
		APIBase:     server.URL,
		OAuthBase:   server.URL,
		ClientID:    "client-id",
		Scopes:      []string{"workers-r2.read"},
		RedirectURI: "http://127.0.0.1:" + port + "/oauth/callback",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	flow, err := client.NewFlow()
	if err != nil {
		t.Fatalf("NewFlow: %v", err)
	}

	go func() {
		callback := fmt.Sprintf("http://127.0.0.1:%s/oauth/callback?code=code-value&state=%s", port, url.QueryEscape(flow.state))
		response, err := http.Get(callback)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tokens, err := flow.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if tokens.AccessToken != "access" || tokens.RefreshToken != "refresh" {
		t.Fatalf("unexpected tokens %+v", tokens)
	}
	if tokens.ExpiresAt.IsZero() {
		t.Fatal("expected an expiry derived from expires_in")
	}
	if form.Get("grant_type") != "authorization_code" || form.Get("code") != "code-value" {
		t.Fatalf("unexpected token request %v", form)
	}
	if form.Get("code_verifier") != flow.verifier || form.Get("client_id") != "client-id" {
		t.Fatalf("unexpected PKCE parameters %v", form)
	}
}

func TestFlowRejectsMismatchedState(t *testing.T) {
	port := freePort(t)
	client, err := New(Config{
		ClientID:    "client-id",
		RedirectURI: "http://127.0.0.1:" + port + "/oauth/callback",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	flow, err := client.NewFlow()
	if err != nil {
		t.Fatalf("NewFlow: %v", err)
	}

	go func() {
		response, err := http.Get("http://127.0.0.1:" + port + "/oauth/callback?code=code-value&state=wrong")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = flow.Wait(ctx)
	if result := core.AsError(err); result == nil || result.Code != "provider_error" {
		t.Fatalf("unexpected error %+v", result)
	}
}

func TestFlowReportsARefusedAuthorization(t *testing.T) {
	port := freePort(t)
	client, err := New(Config{ClientID: "client-id", RedirectURI: "http://127.0.0.1:" + port + "/oauth/callback"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	flow, err := client.NewFlow()
	if err != nil {
		t.Fatalf("NewFlow: %v", err)
	}

	go func() {
		response, err := http.Get("http://127.0.0.1:" + port + "/oauth/callback?error=access_denied&error_description=user+denied")
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = flow.Wait(ctx)
	if result := core.AsError(err); result == nil || result.Code != "permission_denied" {
		t.Fatalf("unexpected error %+v", result)
	}
}

func TestWaitReleasesTheCallbackPort(t *testing.T) {
	port := freePort(t)
	client, err := New(Config{ClientID: "client-id", RedirectURI: "http://127.0.0.1:" + port + "/oauth/callback"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	flow, err := client.NewFlow()
	if err != nil {
		t.Fatalf("NewFlow: %v", err)
	}
	second, err := New(Config{ClientID: "other", RedirectURI: "http://127.0.0.1:" + port + "/oauth/callback"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := second.NewFlow(); err == nil {
		t.Fatal("expected the second sign-in on the same port to be refused")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, _ = flow.Wait(ctx)
	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("the callback port was not released: %v", err)
	}
	_ = listener.Close()
}

func TestExchangeReportsRejectedGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant","error_description":"code expired"}`)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.Exchange(context.Background(), "code", "verifier")
	if result := core.AsError(err); result == nil || result.Code != "credentials_unavailable" {
		t.Fatalf("unexpected error %+v", result)
	}
}

func TestExchangeRequiresAClientID(t *testing.T) {
	client, err := New(Config{RedirectURI: "http://127.0.0.1:8976/oauth/callback"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = client.Exchange(context.Background(), "code", "verifier")
	if result := core.AsError(err); result == nil || result.Code != "configuration" {
		t.Fatalf("unexpected error %+v", result)
	}
}
