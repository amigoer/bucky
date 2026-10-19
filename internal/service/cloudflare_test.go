package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/drivers/cloudflare"
)

const testAccountID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

// memorySecrets stands in for the keychain entries that hold the refresh token.
type memorySecrets struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memorySecrets) GetSecret(key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.values[key]
	if !ok {
		return "", core.NewError("credentials_unavailable", "missing")
	}
	return value, nil
}

func (m *memorySecrets) SetSecret(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
	return nil
}

func (m *memorySecrets) DeleteSecret(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
	return nil
}

func (m *memorySecrets) get(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.values[key]
	return value, ok
}

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

// fakeCloudflare answers the management API and the OAuth endpoints with the
// shapes the driver expects, so the whole sign-in runs offline.
type fakeCloudflare struct {
	server *httptest.Server

	mu       sync.Mutex
	requests []string
	bodies   []map[string]any
	deleted  []string
	minted   int
}

func newFakeCloudflare(t *testing.T) *fakeCloudflare {
	t.Helper()
	f := &fakeCloudflare{}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeCloudflare) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization"))
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/token":
		_ = r.ParseForm()
		if r.PostForm.Get("grant_type") == "refresh_token" {
			_, _ = io.WriteString(w, `{"access_token":"access-refreshed","refresh_token":"refresh-refreshed","token_type":"bearer","expires_in":3600,"scope":"workers-r2.read"}`)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"access-granted","refresh_token":"refresh-granted","token_type":"bearer","expires_in":3600,"scope":"workers-r2.read workers-r2.write"}`)
	case r.URL.Path == "/revoke":
		_, _ = io.WriteString(w, `{}`)
	case r.URL.Path == "/accounts":
		writeResult(w, []cloudflare.Account{{ID: testAccountID, Name: "Example"}})
	case strings.HasSuffix(r.URL.Path, "/r2/buckets"):
		writeResult(w, map[string]any{"buckets": []cloudflare.Bucket{{Name: "media-bucket"}, {Name: "logs"}}})
	case strings.HasSuffix(r.URL.Path, "/tokens/permission_groups"):
		writeResult(w, []cloudflare.PermissionGroup{
			{ID: "group-read", Name: cloudflare.PermissionGroupRead, Scopes: []string{"com.cloudflare.edge.r2.bucket"}},
			{ID: "group-write", Name: cloudflare.PermissionGroupWrite, Scopes: []string{"com.cloudflare.edge.r2.bucket"}},
		})
	case strings.HasSuffix(r.URL.Path, "/tokens") && r.Method == http.MethodPost:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.minted++
		number := f.minted
		f.bodies = append(f.bodies, body)
		f.mu.Unlock()
		writeResult(w, mintedToken(number))
	case strings.Contains(r.URL.Path, "/tokens/") && r.Method == http.MethodDelete:
		f.mu.Lock()
		f.deleted = append(f.deleted, path.Base(r.URL.Path))
		f.mu.Unlock()
		writeResult(w, map[string]any{})
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"success":false,"errors":[{"code":7003,"message":"no route for this path"}]}`)
	}
}

func mintedToken(number int) cloudflare.Token {
	return cloudflare.Token{ID: fmt.Sprintf("token-%d", number), Name: "Bucky media-bucket", Value: fmt.Sprintf("token-value-%d", number)}
}

func (f *fakeCloudflare) seen(needle string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, request := range f.requests {
		if strings.Contains(request, needle) {
			return true
		}
	}
	return false
}

func (f *fakeCloudflare) deletedTokens() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.deleted...)
}

func (f *fakeCloudflare) lastBody() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.bodies) == 0 {
		return nil
	}
	return f.bodies[len(f.bodies)-1]
}

func writeResult(w http.ResponseWriter, result any) {
	data, err := json.Marshal(map[string]any{"success": true, "errors": []any{}, "result": result})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

type cloudflareFixture struct {
	svc     *Service
	vault   *memoryVault
	secrets *memorySecrets
	store   *memoryStore
	fake    *fakeCloudflare
	creds   []core.Credentials
	updates chan CloudflareSignIn
}

// newCloudflareFixture wires the service to the fake Cloudflare endpoints. A
// nil factory records the credentials that reach the S3 driver.
func newCloudflareFixture(t *testing.T, factory core.Factory) *cloudflareFixture {
	t.Helper()
	// Keep the machine's own configuration out of the test.
	t.Setenv("BUCKY_CLOUDFLARE_CLIENT_ID", "")
	t.Setenv("BUCKY_CLOUDFLARE_REDIRECT_URI", "")
	t.Setenv("BUCKY_CLOUDFLARE_SCOPES", "")
	f := &cloudflareFixture{
		vault:   &memoryVault{creds: map[string]core.Credentials{}},
		secrets: &memorySecrets{values: map[string]string{}},
		store:   &memoryStore{objects: map[string]core.ObjectInfo{}},
		fake:    newFakeCloudflare(t),
		updates: make(chan CloudflareSignIn, 8),
	}
	if factory == nil {
		factory = func(_ context.Context, _ core.Profile, c core.Credentials) (core.ObjectStore, error) {
			f.creds = append(f.creds, c)
			return f.store, nil
		}
	}
	redirect := "http://127.0.0.1:" + freePort(t) + "/oauth/callback"
	svc, err := New(context.Background(), Options{
		Dir:     t.TempDir(),
		Vault:   f.vault,
		Secrets: f.secrets,
		Factory: factory,
		Cloudflare: func(cfg cloudflare.Config) (*cloudflare.Client, error) {
			cfg.APIBase = f.fake.server.URL
			cfg.OAuthBase = f.fake.server.URL
			return cloudflare.New(cfg)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	svc.OnCloudflareSignIn = func(update CloudflareSignIn) { f.updates <- update }
	t.Cleanup(func() { _ = svc.Close() })
	if _, err := svc.SaveSettings(context.Background(), Settings{CloudflareClientID: "client-id", CloudflareRedirectURI: redirect}); err != nil {
		t.Fatal(err)
	}
	f.svc = svc
	return f
}

// start begins a sign-in and returns the renderer-visible state.
func (f *cloudflareFixture) start(t *testing.T) CloudflareSignIn {
	t.Helper()
	session, err := f.svc.StartCloudflareSignIn(context.Background())
	if err != nil {
		t.Fatalf("StartCloudflareSignIn: %v", err)
	}
	if session.State != "waiting" || session.SessionID == "" || session.AuthURL == "" {
		t.Fatalf("unexpected session %+v", session)
	}
	return session
}

// authorize plays the browser: it redirects back to the loopback listener with
// the state taken from the authorization URL.
func (f *cloudflareFixture) authorize(t *testing.T, session CloudflareSignIn, code string) {
	t.Helper()
	parsed, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	callback := session.RedirectURI + "?code=" + url.QueryEscape(code) + "&state=" + url.QueryEscape(parsed.Query().Get("state"))
	go func() {
		response, err := http.Get(callback)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()
}

func (f *cloudflareFixture) waitFor(t *testing.T, state string) CloudflareSignIn {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case update := <-f.updates:
			if update.State == state {
				return update
			}
		case <-deadline:
			t.Fatalf("no %q update arrived", state)
		}
	}
}

func (f *cloudflareFixture) signIn(t *testing.T) CloudflareSignIn {
	t.Helper()
	session := f.start(t)
	f.authorize(t, session, "code-granted")
	update := f.waitFor(t, "authorized")
	if len(update.Accounts) != 1 || update.Accounts[0].ID != testAccountID {
		t.Fatalf("unexpected accounts %+v", update.Accounts)
	}
	return update
}

func TestCloudflareSignInMintsBucketScopedCredentials(t *testing.T) {
	f := newCloudflareFixture(t, nil)
	ctx := context.Background()
	session := f.signIn(t)

	if !f.fake.seen("/accounts Bearer access-granted") {
		t.Fatalf("the granted access token was not used: %v", f.fake.requests)
	}
	buckets, err := f.svc.CloudflareBuckets(ctx, session.SessionID, testAccountID)
	if err != nil {
		t.Fatalf("CloudflareBuckets: %v", err)
	}
	if len(buckets) != 2 || buckets[0].Name != "media-bucket" {
		t.Fatalf("buckets = %+v", buckets)
	}

	p, err := f.svc.ConnectCloudflare(ctx, CloudflareConnection{
		SessionID: session.SessionID, ProfileID: "p9", Name: "R2 media",
		AccountID: testAccountID, Bucket: "media-bucket",
	})
	if err != nil {
		t.Fatalf("ConnectCloudflare: %v", err)
	}
	if p.CredentialSource != "cloudflare" || p.Bucket != "media-bucket" || p.ID != "p9" {
		t.Fatalf("profile = %+v", p)
	}

	want := cloudflare.S3Credentials(mintedToken(1))
	stored, err := f.vault.Get("p9")
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	if stored != want {
		t.Fatalf("vault credentials = %+v, want %+v", stored, want)
	}
	if len(f.creds) == 0 || f.creds[len(f.creds)-1] != want {
		t.Fatalf("the S3 driver was not called with the minted credentials: %+v", f.creds)
	}
	if body := f.fake.lastBody(); body == nil {
		t.Fatal("no token was created")
	} else if resource := "com.cloudflare.edge.r2.bucket." + testAccountID + "_default_media-bucket"; !strings.Contains(mustJSON(t, body), resource) {
		t.Fatalf("token body = %s, want resource %q", mustJSON(t, body), resource)
	} else if !strings.Contains(mustJSON(t, body), "group-write") {
		t.Fatalf("token body = %s, want the write permission group", mustJSON(t, body))
	}

	secret, ok := f.secrets.get(cloudflareSecret + "p9")
	if !ok {
		t.Fatal("the authorization was not stored in the keychain")
	}
	var record authorization
	if err := json.Unmarshal([]byte(secret), &record); err != nil {
		t.Fatalf("stored authorization: %v", err)
	}
	if record.RefreshToken != "refresh-granted" || record.TokenID != "token-1" || record.AccountID != testAccountID {
		t.Fatalf("stored authorization = %+v", record)
	}
	// The refresh token belongs to the keychain only, never to the database.
	if raw := mustJSON(t, mustProfile(t, f.svc, "p9")); strings.Contains(raw, "refresh-granted") || strings.Contains(raw, "token-value-1") {
		t.Fatalf("the database profile leaked a secret: %s", raw)
	}

	status, err := f.svc.CloudflareStatus(ctx, "p9")
	if err != nil {
		t.Fatalf("CloudflareStatus: %v", err)
	}
	if !status.Authorized || status.TokenName != "Bucky media-bucket" {
		t.Fatalf("status = %+v", status)
	}
	// A completed sign-in is consumed by the connection.
	if _, err := f.svc.CloudflareBuckets(ctx, session.SessionID, testAccountID); core.AsError(err).Code != "not_found" {
		t.Fatalf("the sign-in session outlived the connection: %v", err)
	}

	if err := f.svc.DisconnectCloudflare(ctx, "p9", false); core.AsError(err).Code != "confirmation_required" {
		t.Fatalf("unconfirmed disconnect = %v", err)
	}
	if err := f.svc.DisconnectCloudflare(ctx, "p9", true); err != nil {
		t.Fatalf("DisconnectCloudflare: %v", err)
	}
	if deleted := f.fake.deletedTokens(); len(deleted) != 1 || deleted[0] != "token-1" {
		t.Fatalf("deleted tokens = %v", deleted)
	}
	if !f.fake.seen("/revoke") {
		t.Fatal("the refresh token was not revoked")
	}
	if _, err := f.vault.Get("p9"); core.AsError(err).Code != "credentials_unavailable" {
		t.Fatalf("the vault entry survived the disconnect: %v", err)
	}
	if _, ok := f.secrets.get(cloudflareSecret + "p9"); ok {
		t.Fatal("the keychain authorization survived the disconnect")
	}
}

func TestCloudflareReadOnlyConnectionMintsTheReadGroup(t *testing.T) {
	f := newCloudflareFixture(t, nil)
	session := f.signIn(t)
	if _, err := f.svc.ConnectCloudflare(context.Background(), CloudflareConnection{
		SessionID: session.SessionID, ProfileID: "p9", Name: "R2 media",
		AccountID: testAccountID, Bucket: "media-bucket", ReadOnly: true,
	}); err != nil {
		t.Fatalf("ConnectCloudflare: %v", err)
	}
	body := mustJSON(t, f.fake.lastBody())
	if !strings.Contains(body, "group-read") || strings.Contains(body, "group-write") {
		t.Fatalf("token body = %s, want the read permission group", body)
	}
}

func TestCloudflareConnectionKeepsNoTokenWhenVerificationFails(t *testing.T) {
	failing := func(context.Context, core.Profile, core.Credentials) (core.ObjectStore, error) {
		return nil, core.NewError("credentials_unavailable", "The minted R2 token was rejected")
	}
	f := newCloudflareFixture(t, failing)
	session := f.signIn(t)
	_, err := f.svc.ConnectCloudflare(context.Background(), CloudflareConnection{
		SessionID: session.SessionID, ProfileID: "p9", Name: "R2 media",
		AccountID: testAccountID, Bucket: "media-bucket",
	})
	if result := core.AsError(err); result == nil || result.Code != "credentials_unavailable" {
		t.Fatalf("unexpected error %+v", result)
	}
	if deleted := f.fake.deletedTokens(); len(deleted) != 1 || deleted[0] != "token-1" {
		t.Fatalf("the rejected token was left behind: %v", deleted)
	}
	if profiles, err := f.svc.DB.Profiles(context.Background()); err != nil || len(profiles) != 0 {
		t.Fatalf("profiles = %+v, err = %v", profiles, err)
	}
	if _, ok := f.secrets.get(cloudflareSecret + "p9"); ok {
		t.Fatal("a failed connection stored an authorization")
	}
}

func TestReauthorizeReplacesTheMintedToken(t *testing.T) {
	f := newCloudflareFixture(t, nil)
	ctx := context.Background()
	session := f.signIn(t)
	if _, err := f.svc.ConnectCloudflare(ctx, CloudflareConnection{
		SessionID: session.SessionID, ProfileID: "p9", Name: "R2 media",
		AccountID: testAccountID, Bucket: "media-bucket",
	}); err != nil {
		t.Fatalf("ConnectCloudflare: %v", err)
	}
	if _, err := f.svc.ReauthorizeCloudflare(ctx, "p9"); err != nil {
		t.Fatalf("ReauthorizeCloudflare: %v", err)
	}
	want := cloudflare.S3Credentials(mintedToken(2))
	if stored, err := f.vault.Get("p9"); err != nil || stored != want {
		t.Fatalf("vault credentials = %+v, err = %v", stored, err)
	}
	if deleted := f.fake.deletedTokens(); len(deleted) != 1 || deleted[0] != "token-1" {
		t.Fatalf("the replaced token was not removed: %v", deleted)
	}
	secret, ok := f.secrets.get(cloudflareSecret + "p9")
	if !ok {
		t.Fatal("the refreshed authorization was not stored")
	}
	var record authorization
	if err := json.Unmarshal([]byte(secret), &record); err != nil {
		t.Fatalf("stored authorization: %v", err)
	}
	if record.TokenID != "token-2" || record.RefreshToken != "refresh-refreshed" {
		t.Fatalf("stored authorization = %+v", record)
	}
}

func TestCloudflareSignInCanStartOverAfterAFailedConnection(t *testing.T) {
	var store *memoryStore
	reject := true
	factory := func(context.Context, core.Profile, core.Credentials) (core.ObjectStore, error) {
		if reject {
			return nil, core.NewError("credentials_unavailable", "The minted R2 token was rejected")
		}
		return store, nil
	}
	f := newCloudflareFixture(t, factory)
	store = f.store
	ctx := context.Background()

	session := f.signIn(t)
	if _, err := f.svc.ConnectCloudflare(ctx, CloudflareConnection{
		SessionID: session.SessionID, ProfileID: "p9", Name: "R2 media",
		AccountID: testAccountID, Bucket: "media-bucket",
	}); core.AsError(err).Code != "credentials_unavailable" {
		t.Fatalf("err = %v", err)
	}

	// The finished session must not block a new attempt with the same port.
	reject = false
	second, err := f.svc.StartCloudflareSignIn(ctx)
	if err != nil {
		t.Fatalf("StartCloudflareSignIn: %v", err)
	}
	if second.SessionID == session.SessionID || second.State != "waiting" {
		t.Fatalf("unexpected session %+v", second)
	}
	f.authorize(t, second, "code-granted")
	update := f.waitFor(t, "authorized")
	if _, err := f.svc.ConnectCloudflare(ctx, CloudflareConnection{
		SessionID: update.SessionID, ProfileID: "p9", Name: "R2 media",
		AccountID: testAccountID, Bucket: "media-bucket",
	}); err != nil {
		t.Fatalf("ConnectCloudflare: %v", err)
	}
	if _, err := f.vault.Get("p9"); err != nil {
		t.Fatalf("vault: %v", err)
	}
}

func TestCloudflareSignInRequiresAClientID(t *testing.T) {
	f := newCloudflareFixture(t, nil)
	if _, err := f.svc.SaveSettings(context.Background(), Settings{CloudflareClientID: "", CloudflareRedirectURI: "http://127.0.0.1:8976/oauth/callback"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.StartCloudflareSignIn(context.Background()); core.AsError(err).Code != "configuration" {
		t.Fatalf("err = %v", err)
	}
}

func TestCloudflareSignInRefusesAConcurrentAttempt(t *testing.T) {
	f := newCloudflareFixture(t, nil)
	first := f.start(t)
	if _, err := f.svc.StartCloudflareSignIn(context.Background()); core.AsError(err).Code != "conflict" {
		t.Fatalf("err = %v", err)
	}
	if err := f.svc.CancelCloudflareSignIn(context.Background(), first.SessionID); err != nil {
		t.Fatalf("CancelCloudflareSignIn: %v", err)
	}
	update := f.waitFor(t, "failed")
	if update.Error == nil || update.Error.Code != "cancelled" {
		t.Fatalf("update = %+v", update)
	}
	if _, err := f.svc.CloudflareSignInStatus(context.Background(), first.SessionID); core.AsError(err).Code != "not_found" {
		t.Fatalf("the cancelled session is still listed: %v", err)
	}
	// A cancelled sign-in must leave the callback port free.
	parsed, err := url.Parse(first.RedirectURI)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		listener, err := net.Listen("tcp", parsed.Host)
		if err == nil {
			_ = listener.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the callback port was not released: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCloudflareSignInReportsARefusedAuthorization(t *testing.T) {
	f := newCloudflareFixture(t, nil)
	session := f.start(t)
	parsed, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatal(err)
	}
	callback := session.RedirectURI + "?error=access_denied&error_description=user+denied&state=" + url.QueryEscape(parsed.Query().Get("state"))
	go func() {
		response, err := http.Get(callback)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()
	update := f.waitFor(t, "failed")
	if update.Error == nil || update.Error.Code != "permission_denied" {
		t.Fatalf("update = %+v", update)
	}
}

func TestSettingsRoundTripAndValidation(t *testing.T) {
	f := newCloudflareFixture(t, nil)
	ctx := context.Background()

	saved, err := f.svc.SaveSettings(ctx, Settings{
		CloudflareClientID:    "  client-id  ",
		CloudflareRedirectURI: "",
		CloudflareScopes:      []string{"workers-r2.read", "workers-r2.read", "not a scope"},
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if saved.CloudflareClientID != "client-id" || saved.CloudflareRedirectURI != defaultCloudflareRedirect {
		t.Fatalf("saved = %+v", saved)
	}
	if len(saved.CloudflareScopes) != 1 || saved.CloudflareScopes[0] != "workers-r2.read" {
		t.Fatalf("scopes = %v", saved.CloudflareScopes)
	}
	loaded, err := f.svc.Settings(ctx)
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if loaded.CloudflareClientID != "client-id" || loaded.CloudflareRedirectURI != defaultCloudflareRedirect {
		t.Fatalf("loaded = %+v", loaded)
	}
	if _, err := f.svc.SaveSettings(ctx, Settings{CloudflareClientID: "bad id", CloudflareRedirectURI: defaultCloudflareRedirect}); core.AsError(err).Code != "invalid_argument" {
		t.Fatalf("err = %v", err)
	}
	if _, err := f.svc.SaveSettings(ctx, Settings{CloudflareClientID: "client-id", CloudflareRedirectURI: "https://example.com/callback"}); core.AsError(err).Code != "invalid_argument" {
		t.Fatalf("err = %v", err)
	}
}

func TestCloudflareProfilesKeepTheirCredentialsOutOfTheInput(t *testing.T) {
	f := newCloudflareFixture(t, nil)
	ctx := context.Background()
	// Seed a connection the way the sign-in flow stores one.
	minted := core.Credentials{AccessKeyID: "minted", SecretAccessKey: "minted-secret"}
	if err := f.vault.Set("p9", minted); err != nil {
		t.Fatal(err)
	}
	if err := f.secrets.SetSecret(cloudflareSecret+"p9", `{"refreshToken":"refresh","tokenId":"token-1"}`); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DB.SaveProfile(ctx, core.Profile{ID: "p9", Name: "R2 media", AccountID: testAccountID, Bucket: "media-bucket", CredentialSource: "cloudflare"}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.SaveProfile(ctx, ProfileInput{
		Profile:     core.Profile{ID: "p9", Name: "R2 media", AccountID: testAccountID, Bucket: "media-bucket", CredentialSource: "cloudflare"},
		AccessKeyID: "key", SecretAccessKey: "secret",
	}); core.AsError(err).Code != "invalid_argument" {
		t.Fatalf("err = %v", err)
	}
	if _, err := f.svc.SaveProfile(ctx, ProfileInput{
		Profile: core.Profile{ID: "p9", Name: "R2 media renamed", AccountID: testAccountID, Bucket: "media-bucket", CredentialSource: "cloudflare"},
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if stored, err := f.vault.Get("p9"); err != nil || stored != minted {
		t.Fatalf("the minted credentials were replaced: %+v, err = %v", stored, err)
	}

	if err := f.svc.RemoveProfile(ctx, "p9", true); err != nil {
		t.Fatalf("RemoveProfile: %v", err)
	}
	if _, err := f.vault.Get("p9"); core.AsError(err).Code != "credentials_unavailable" {
		t.Fatalf("the vault entry survived removal: %v", err)
	}
	if _, ok := f.secrets.get(cloudflareSecret + "p9"); ok {
		t.Fatal("the keychain authorization survived removal")
	}

	if _, err := f.svc.ReauthorizeCloudflare(ctx, "missing"); core.AsError(err).Code != "not_found" {
		t.Fatalf("err = %v", err)
	}
	if _, err := f.svc.ConnectCloudflare(ctx, CloudflareConnection{SessionID: "gone", Name: "R2 media", AccountID: testAccountID, Bucket: "media-bucket"}); core.AsError(err).Code != "not_found" {
		t.Fatalf("err = %v", err)
	}
}

func mustProfile(t *testing.T, svc *Service, id string) core.Profile {
	t.Helper()
	p, err := svc.DB.Profile(context.Background(), id)
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	return p
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(data)
}
