package cloudflare

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/amigoer/bucky/internal/core"
)

// Tokens is one OAuth token response. The refresh token is the only value that
// survives a session; it is stored in the system keychain by the caller.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	ExpiresAt    time.Time
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (t tokenResponse) tokens() Tokens {
	result := Tokens{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, TokenType: t.TokenType, Scope: t.Scope}
	if t.ExpiresIn > 0 {
		result.ExpiresAt = time.Now().UTC().Add(time.Duration(t.ExpiresIn) * time.Second)
	}
	return result
}

func (c *Client) Exchange(ctx context.Context, code, verifier string) (Tokens, error) {
	if c.cfg.ClientID == "" {
		return Tokens{}, core.NewError("configuration", "Set the Cloudflare OAuth client ID in settings first")
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.cfg.RedirectURI},
		"client_id":     {c.cfg.ClientID},
		"code_verifier": {verifier},
	}
	response, err := c.postForm(ctx, "/token", form)
	if err != nil {
		return Tokens{}, err
	}
	if response.AccessToken == "" {
		return Tokens{}, core.NewError("provider_error", "Cloudflare returned no access token")
	}
	return response.tokens(), nil
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	if refreshToken == "" {
		return Tokens{}, core.NewError("credentials_unavailable", "This connection has no stored Cloudflare authorization; sign in again")
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.cfg.ClientID},
	}
	response, err := c.postForm(ctx, "/token", form)
	if err != nil {
		return Tokens{}, err
	}
	if response.AccessToken == "" {
		return Tokens{}, core.NewError("provider_error", "Cloudflare returned no access token")
	}
	return response.tokens(), nil
}

func (c *Client) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	form := url.Values{"token": {token}, "client_id": {c.cfg.ClientID}}
	_, err := c.postForm(ctx, "/revoke", form)
	return err
}

func (c *Client) postForm(ctx context.Context, path string, form url.Values) (tokenResponse, error) {
	if err := ctx.Err(); err != nil {
		return tokenResponse{}, core.AsError(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.OAuthBase+path, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, core.NewError("internal", "Cannot build the Cloudflare request")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	if c.cfg.UserAgent != "" {
		request.Header.Set("User-Agent", c.cfg.UserAgent)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return tokenResponse{}, transportError(err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBody))
	if err != nil {
		return tokenResponse{}, transportError(err)
	}
	if response.StatusCode >= 400 {
		return tokenResponse{}, oauthError(response, data)
	}
	var tokens tokenResponse
	if err := json.Unmarshal(data, &tokens); err != nil {
		return tokenResponse{}, core.NewError("provider_error", "The Cloudflare token endpoint returned an invalid response")
	}
	return tokens, nil
}

func oauthError(response *http.Response, data []byte) *core.Error {
	var parsed struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	_ = json.Unmarshal(data, &parsed)
	result := codeForStatus(response.StatusCode)
	switch parsed.Error {
	case "invalid_grant":
		result = core.NewError("credentials_unavailable", "Cloudflare rejected the authorization code or refresh token; sign in again")
	case "invalid_client":
		result = core.NewError("configuration", "Cloudflare does not recognize this OAuth client ID")
	case "unauthorized_client":
		result = core.NewError("configuration", "This OAuth client may not use the authorization code flow")
	case "invalid_request":
		result = core.NewError("invalid_argument", "Cloudflare rejected the authorization request")
	}
	if parsed.Description != "" {
		result.Message = withDetail(result.Message, parsed.Description)
	}
	return result
}

type loopback struct {
	host string
	port string
	path string
}

func parseLoopback(raw string) (loopback, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !isLoopbackHost(u.Hostname()) {
		return loopback{}, core.Invalid("The OAuth redirect URI must be a loopback HTTP address such as http://127.0.0.1:8976/oauth/callback")
	}
	if u.Port() == "" {
		return loopback{}, core.Invalid("The OAuth redirect URI must include a port")
	}
	path := u.Path
	if path == "" {
		path = "/"
	}
	return loopback{host: u.Hostname(), port: u.Port(), path: path}, nil
}

type flowResult struct {
	code string
	err  error
}

// Flow serves the single OAuth redirect that completes a sign-in.
type Flow struct {
	client   *Client
	server   *http.Server
	listener net.Listener
	redirect string
	state    string
	verifier string
	result   chan flowResult
	finish   sync.Once
	shutdown sync.Once
}

func (c *Client) NewFlow() (*Flow, error) {
	target, err := parseLoopback(c.cfg.RedirectURI)
	if err != nil {
		return nil, err
	}
	state, err := randomValue(16)
	if err != nil {
		return nil, err
	}
	verifier, err := randomValue(32)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(target.host, target.port))
	if err != nil {
		return nil, core.NewError("configuration", fmt.Sprintf("The sign-in callback address %s is not available; close whatever is using that port", c.cfg.RedirectURI))
	}
	flow := &Flow{
		client:   c,
		listener: listener,
		redirect: c.cfg.RedirectURI,
		state:    state,
		verifier: verifier,
		result:   make(chan flowResult, 1),
	}
	mux := http.NewServeMux()
	mux.HandleFunc(target.path, flow.handle)
	flow.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = flow.server.Serve(listener) }()
	return flow, nil
}

func (f *Flow) RedirectURI() string { return f.redirect }

func (f *Flow) AuthURL() string {
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {f.client.cfg.ClientID},
		"redirect_uri":          {f.redirect},
		"scope":                 {strings.Join(f.client.cfg.Scopes, " ")},
		"state":                 {f.state},
		"code_challenge":        {codeChallenge(f.verifier)},
		"code_challenge_method": {"S256"},
	}
	return f.client.cfg.OAuthBase + "/auth?" + query.Encode()
}

// Wait blocks until Cloudflare redirects back, then exchanges the code. It
// always releases the listener, so a cancelled sign-in leaves no port open.
func (f *Flow) Wait(ctx context.Context) (Tokens, error) {
	defer func() { _ = f.Close() }()
	select {
	case <-ctx.Done():
		return Tokens{}, core.AsError(ctx.Err())
	case result := <-f.result:
		if result.err != nil {
			return Tokens{}, result.err
		}
		return f.client.Exchange(ctx, result.code, f.verifier)
	}
}

func (f *Flow) Close() error {
	var err error
	f.shutdown.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err = f.server.Shutdown(ctx)
	})
	return err
}

func (f *Flow) handle(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if denied := query.Get("error"); denied != "" {
		f.complete(flowResult{err: core.NewError("permission_denied", withDetail("Cloudflare authorization was refused", query.Get("error_description")))}, w, false)
		return
	}
	if query.Get("state") != f.state {
		f.complete(flowResult{err: core.NewError("provider_error", "The Cloudflare callback did not match this sign-in attempt")}, w, false)
		return
	}
	code := query.Get("code")
	if code == "" {
		f.complete(flowResult{err: core.NewError("provider_error", "The Cloudflare callback carried no authorization code")}, w, false)
		return
	}
	f.complete(flowResult{code: code}, w, true)
}

func (f *Flow) complete(result flowResult, w http.ResponseWriter, ok bool) {
	f.finish.Do(func() { f.result <- result })
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
	}
	_, _ = io.WriteString(w, callbackPage(ok))
}

func callbackPage(ok bool) string {
	title, body := "Authorization complete", "You can close this tab and return to Bucky."
	if !ok {
		title, body = "Authorization failed", "Return to Bucky and try again."
	}
	return "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>" + title +
		"</title><style>body{margin:0;display:flex;min-height:100vh;align-items:center;justify-content:center;" +
		"font:15px/1.6 -apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;color:#1e2939;background:#f7f8fa}" +
		"div{text-align:center}h1{margin:0 0 6px;font-size:17px}p{margin:0;color:#667085}</style></head><body><div><h1>" +
		title + "</h1><p>" + body + "</p></div></body></html>"
}

func randomValue(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", core.NewError("internal", "Cannot read the secure random source")
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
