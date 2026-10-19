// Package cloudflare calls the Cloudflare management API and its OAuth
// endpoints. It never touches object storage: R2 objects are only reachable
// with an R2 API token, which this package mints from an OAuth authorization.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/amigoer/bucky/internal/core"
)

const (
	defaultAPIBase   = "https://api.cloudflare.com/client/v4"
	defaultOAuthBase = "https://dash.cloudflare.com/oauth2"
	defaultRedirect  = "http://127.0.0.1:8976/oauth/callback"
	maxBody          = 1 << 20
	maxDetail        = 200
)

type Config struct {
	APIBase     string
	OAuthBase   string
	ClientID    string
	Scopes      []string
	RedirectURI string
	UserAgent   string
	Timeout     time.Duration
	HTTPClient  *http.Client
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) (*Client, error) {
	apiBase, err := origin(cfg.APIBase, defaultAPIBase)
	if err != nil {
		return nil, core.Invalid("The Cloudflare API base URL must be an HTTPS origin")
	}
	oauthBase, err := origin(cfg.OAuthBase, defaultOAuthBase)
	if err != nil {
		return nil, core.Invalid("The Cloudflare OAuth base URL must be an HTTPS origin")
	}
	if cfg.RedirectURI == "" {
		cfg.RedirectURI = defaultRedirect
	}
	if _, err := parseLoopback(cfg.RedirectURI); err != nil {
		return nil, err
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Timeout < 0 {
		return nil, core.Invalid("The Cloudflare request timeout must be positive")
	}
	cfg.APIBase, cfg.OAuthBase = apiBase, oauthBase
	client := cfg.HTTPClient
	if client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.DialContext = (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
		transport.TLSHandshakeTimeout = 10 * time.Second
		transport.ResponseHeaderTimeout = 30 * time.Second
		client = &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	}
	return &Client{cfg: cfg, http: client}, nil
}

func (c *Client) Config() Config { return c.cfg }

// origin accepts only an absolute HTTP(S) origin; plain HTTP is allowed for
// loopback so tests can point the client at a local server.
func origin(raw, fallback string) (string, error) {
	if raw == "" {
		return fallback, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return "", core.Invalid("Invalid Cloudflare base URL")
	}
	if u.Scheme == "https" {
		return strings.TrimSuffix(u.String(), "/"), nil
	}
	if u.Scheme != "http" || !isLoopbackHost(u.Hostname()) {
		return "", core.Invalid("Invalid Cloudflare base URL")
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}

func isLoopbackHost(host string) bool {
	switch host {
	case "localhost", "::1":
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Success bool            `json:"success"`
	Errors  []apiError      `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

func (c *Client) do(ctx context.Context, token, method, path string, body, out any, headers map[string]string) error {
	if err := ctx.Err(); err != nil {
		return core.AsError(err)
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return core.NewError("internal", "Cannot encode the Cloudflare request")
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.cfg.APIBase+path, reader)
	if err != nil {
		return core.NewError("internal", "Cannot build the Cloudflare request")
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if c.cfg.UserAgent != "" {
		request.Header.Set("User-Agent", c.cfg.UserAgent)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return transportError(err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBody))
	if err != nil {
		return transportError(err)
	}
	var parsed envelope
	unmarshalErr := json.Unmarshal(data, &parsed)
	if response.StatusCode >= 400 {
		return statusError(response, parsed)
	}
	if unmarshalErr != nil {
		return core.NewError("provider_error", "The Cloudflare API returned an invalid response")
	}
	if !parsed.Success {
		return apiFailure(parsed.Errors)
	}
	if out == nil || len(parsed.Result) == 0 || string(parsed.Result) == "null" {
		return nil
	}
	if err := json.Unmarshal(parsed.Result, out); err != nil {
		return core.NewError("provider_error", "The Cloudflare API returned an unexpected result")
	}
	return nil
}

func apiFailure(list []apiError) *core.Error {
	result := codeForStatus(0)
	if len(list) > 0 {
		result.Message = withDetail(result.Message, list[0].Message)
	}
	return result
}

func statusError(response *http.Response, parsed envelope) *core.Error {
	result := codeForStatus(response.StatusCode)
	if len(parsed.Errors) > 0 {
		result.Message = withDetail(result.Message, parsed.Errors[0].Message)
	}
	result.RequestID = safeToken(response.Header.Get("cf-ray"))
	return result
}

func codeForStatus(status int) *core.Error {
	switch {
	case status == 400:
		return core.NewError("invalid_argument", "Cloudflare rejected the request")
	case status == 401:
		return core.NewError("credentials_unavailable", "Cloudflare rejected the sign-in; authorize again")
	case status == 403:
		return core.NewError("permission_denied", "The Cloudflare authorization does not cover this action")
	case status == 404:
		return core.NewError("not_found", "The Cloudflare resource does not exist")
	case status == 409:
		return core.NewError("conflict", "Cloudflare reported a conflicting state")
	case status == 429:
		return &core.Error{Code: "rate_limited", Message: "Cloudflare is rate limiting this client", Retryable: true}
	case status >= 500:
		return &core.Error{Code: "unavailable", Message: "The Cloudflare API is temporarily unavailable", Retryable: true}
	default:
		return core.NewError("provider_error", "The Cloudflare API request failed")
	}
}

func transportError(err error) *core.Error {
	if errors.Is(err, context.Canceled) {
		return core.NewError("cancelled", "Operation cancelled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &core.Error{Code: "timeout", Message: "The Cloudflare request timed out", Retryable: true}
	}
	var network net.Error
	if errors.As(err, &network) {
		if network.Timeout() {
			return &core.Error{Code: "timeout", Message: "The Cloudflare request timed out", Retryable: true}
		}
		return &core.Error{Code: "network", Message: "The Cloudflare API could not be reached", Retryable: true}
	}
	return core.NewError("network", "The Cloudflare API could not be reached")
}

// withDetail keeps the mapped message and appends the provider explanation.
func withDetail(base, detail string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, strings.TrimSpace(detail))
	runes := []rune(cleaned)
	if len(runes) == 0 {
		return base
	}
	if len(runes) > maxDetail {
		runes = runes[:maxDetail]
	}
	return base + ": " + string(runes)
}

func safeToken(value string) string {
	if len(value) > 64 {
		return ""
	}
	for _, c := range value {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && !strings.ContainsRune("-_.:/+=", c) {
			return ""
		}
	}
	return value
}
