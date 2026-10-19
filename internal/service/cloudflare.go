package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/drivers/cloudflare"
)

const (
	settingsKey       = "settings"
	cloudflareTimeout = 5 * time.Minute
	cloudflareSecret  = "cloudflare-oauth:"
	userAgent         = "Bucky"
)

// Settings holds the non-secret configuration of the desktop application.
type Settings struct {
	CloudflareClientID    string   `json:"cloudflareClientId"`
	CloudflareRedirectURI string   `json:"cloudflareRedirectUri"`
	CloudflareScopes      []string `json:"cloudflareScopes"`
}

// Scope names follow the Cloudflare permission names shown on the consent
// screen; the exact identifiers are confirmed against the dashboard.
var defaultCloudflareScopes = []string{
	"account-settings.read",
	"workers-r2.read",
	"workers-r2.write",
	"account-api-tokens.write",
}

const defaultCloudflareRedirect = "http://127.0.0.1:8976/oauth/callback"

type CloudflareAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CloudflareSignIn is the renderer-visible state of a pending sign-in. It never
// carries an access token.
type CloudflareSignIn struct {
	SessionID   string              `json:"sessionId"`
	State       string              `json:"state"`
	AuthURL     string              `json:"authUrl"`
	RedirectURI string              `json:"redirectUri"`
	Accounts    []CloudflareAccount `json:"accounts"`
	Error       *core.Error         `json:"error,omitempty"`
}

type CloudflareConnection struct {
	SessionID    string `json:"sessionId"`
	ProfileID    string `json:"profileId"`
	Name         string `json:"name"`
	AccountID    string `json:"accountId"`
	Bucket       string `json:"bucket"`
	Jurisdiction string `json:"jurisdiction"`
	PublicURL    string `json:"publicUrl"`
	ReadOnly     bool   `json:"readOnly"`
}

type CloudflareStatus struct {
	Authorized   bool      `json:"authorized"`
	TokenName    string    `json:"tokenName"`
	AuthorizedAt time.Time `json:"authorizedAt"`
}

// authorization is what survives a sign-in. It is kept in the system keychain
// and never reaches the database or the renderer.
type authorization struct {
	RefreshToken string    `json:"refreshToken"`
	AccountID    string    `json:"accountId"`
	TokenID      string    `json:"tokenId"`
	TokenName    string    `json:"tokenName"`
	Jurisdiction string    `json:"jurisdiction"`
	ReadOnly     bool      `json:"readOnly"`
	Scope        string    `json:"scope"`
	AuthorizedAt time.Time `json:"authorizedAt"`
}

type signInSession struct {
	id     string
	client *cloudflare.Client
	flow   *cloudflare.Flow
	notify func(CloudflareSignIn)

	mu       sync.Mutex
	status   CloudflareSignIn
	tokens   cloudflare.Tokens
	cancel   context.CancelFunc
	finished bool
}

func (s *Service) Settings(ctx context.Context) (Settings, error) {
	result := Settings{CloudflareRedirectURI: defaultCloudflareRedirect, CloudflareScopes: append([]string{}, defaultCloudflareScopes...)}
	switch err := s.DB.Setting(ctx, settingsKey, &result); {
	case err == nil:
	case core.AsError(err).Code == "not_found":
	default:
		return result, err
	}
	if value := os.Getenv("BUCKY_CLOUDFLARE_CLIENT_ID"); value != "" {
		result.CloudflareClientID = value
	}
	if value := os.Getenv("BUCKY_CLOUDFLARE_REDIRECT_URI"); value != "" {
		result.CloudflareRedirectURI = value
	}
	if value := os.Getenv("BUCKY_CLOUDFLARE_SCOPES"); value != "" {
		result.CloudflareScopes = splitScopes(value)
	}
	if len(result.CloudflareScopes) == 0 {
		result.CloudflareScopes = append([]string{}, defaultCloudflareScopes...)
	}
	return result, nil
}

func (s *Service) SaveSettings(ctx context.Context, input Settings) (Settings, error) {
	input.CloudflareClientID = strings.TrimSpace(input.CloudflareClientID)
	input.CloudflareRedirectURI = strings.TrimSpace(input.CloudflareRedirectURI)
	if input.CloudflareRedirectURI == "" {
		input.CloudflareRedirectURI = defaultCloudflareRedirect
	}
	if input.CloudflareClientID != "" && len(input.CloudflareClientID) > 64 {
		return input, core.Invalid("The Cloudflare OAuth client ID is too long")
	}
	if input.CloudflareClientID != "" && strings.ContainsAny(input.CloudflareClientID, " \t\r\n") {
		return input, core.Invalid("The Cloudflare OAuth client ID cannot contain spaces")
	}
	input.CloudflareScopes = cleanScopes(input.CloudflareScopes)
	if len(input.CloudflareScopes) == 0 {
		input.CloudflareScopes = append([]string{}, defaultCloudflareScopes...)
	}
	// The redirect URI has to match the one registered with Cloudflare, so it is
	// validated by building a client with it.
	if _, err := s.cloudflare(cloudflare.Config{
		ClientID:    input.CloudflareClientID,
		Scopes:      input.CloudflareScopes,
		RedirectURI: input.CloudflareRedirectURI,
		UserAgent:   userAgent,
	}); err != nil {
		return input, err
	}
	if err := s.DB.SaveSetting(ctx, settingsKey, input); err != nil {
		return input, err
	}
	return input, nil
}

func (s *Service) cloudflareClient(ctx context.Context) (*cloudflare.Client, Settings, error) {
	settings, err := s.Settings(ctx)
	if err != nil {
		return nil, settings, err
	}
	if settings.CloudflareClientID == "" {
		return nil, settings, core.NewError("configuration", "Add the Cloudflare OAuth client ID in settings before signing in")
	}
	client, err := s.cloudflare(cloudflare.Config{
		ClientID:    settings.CloudflareClientID,
		Scopes:      settings.CloudflareScopes,
		RedirectURI: settings.CloudflareRedirectURI,
		UserAgent:   userAgent,
	})
	if err != nil {
		return nil, settings, err
	}
	return client, settings, nil
}

func (s *Service) StartCloudflareSignIn(ctx context.Context) (CloudflareSignIn, error) {
	client, settings, err := s.cloudflareClient(ctx)
	if err != nil {
		return CloudflareSignIn{}, err
	}
	// The callback port is fixed, so only one sign-in can be pending. A finished
	// session is dropped here: the renderer may start over after a cancelled or
	// failed connection attempt, and its session is no longer usable.
	s.signInMu.Lock()
	for id, session := range s.signIns {
		if session.snapshot().State == "waiting" {
			s.signInMu.Unlock()
			return CloudflareSignIn{}, core.NewError("conflict", "A Cloudflare sign-in is already waiting for authorization")
		}
		delete(s.signIns, id)
	}
	s.signInMu.Unlock()
	flow, err := client.NewFlow()
	if err != nil {
		return CloudflareSignIn{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cloudflareTimeout)
	session := &signInSession{
		id:     ID(),
		client: client,
		flow:   flow,
		notify: s.OnCloudflareSignIn,
		cancel: cancel,
		status: CloudflareSignIn{
			SessionID:   "",
			State:       "waiting",
			AuthURL:     flow.AuthURL(),
			RedirectURI: settings.CloudflareRedirectURI,
			Accounts:    []CloudflareAccount{},
		},
	}
	session.status.SessionID = session.id
	s.signInMu.Lock()
	for _, other := range s.signIns {
		if other.snapshot().State == "waiting" {
			s.signInMu.Unlock()
			cancel()
			_ = flow.Close()
			return CloudflareSignIn{}, core.NewError("conflict", "A Cloudflare sign-in is already waiting for authorization")
		}
	}
	if s.signIns == nil {
		s.signIns = map[string]*signInSession{}
	}
	s.signIns[session.id] = session
	s.signInMu.Unlock()

	go s.runCloudflareSignIn(ctx, session)
	return session.snapshot(), nil
}

func (s *Service) runCloudflareSignIn(ctx context.Context, session *signInSession) {
	defer session.cancel()
	tokens, err := session.flow.Wait(ctx)
	if err != nil {
		session.finish("failed", nil, err)
		s.releaseSignIn(session.id)
		return
	}
	accounts, err := session.client.Accounts(ctx, tokens.AccessToken)
	if err != nil {
		session.finish("failed", nil, err)
		s.releaseSignIn(session.id)
		return
	}
	list := make([]CloudflareAccount, 0, len(accounts))
	for _, account := range accounts {
		list = append(list, CloudflareAccount{ID: account.ID, Name: account.Name})
	}
	if len(list) == 0 {
		session.finish("failed", nil, core.NewError("permission_denied", "This authorization covers no Cloudflare account"))
		s.releaseSignIn(session.id)
		return
	}
	session.mu.Lock()
	session.tokens = tokens
	session.mu.Unlock()
	session.finish("authorized", list, nil)
}

// finish reports the terminal state once; later calls keep the first result.
func (s *signInSession) finish(state string, accounts []CloudflareAccount, err error) {
	s.mu.Lock()
	if s.finished {
		s.mu.Unlock()
		return
	}
	s.finished = true
	s.status.State = state
	s.status.Accounts = accounts
	s.status.Error = core.AsError(err)
	if s.status.Accounts == nil {
		s.status.Accounts = []CloudflareAccount{}
	}
	update := s.status
	s.mu.Unlock()
	if s.notify != nil {
		s.notify(update)
	}
}

func (s *signInSession) snapshot() CloudflareSignIn {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.status
	if result.Accounts == nil {
		result.Accounts = []CloudflareAccount{}
	}
	return result
}

func (s *Service) releaseSignIn(id string) {
	s.signInMu.Lock()
	delete(s.signIns, id)
	s.signInMu.Unlock()
}

func (s *Service) CloudflareSignInStatus(ctx context.Context, sessionID string) (CloudflareSignIn, error) {
	s.signInMu.Lock()
	session, ok := s.signIns[sessionID]
	s.signInMu.Unlock()
	if !ok {
		return CloudflareSignIn{}, core.NewError("not_found", "This sign-in attempt is no longer available")
	}
	return session.snapshot(), nil
}

func (s *Service) CancelCloudflareSignIn(ctx context.Context, sessionID string) error {
	s.signInMu.Lock()
	session, ok := s.signIns[sessionID]
	s.signInMu.Unlock()
	if !ok {
		return nil
	}
	session.mu.Lock()
	cancel := session.cancel
	session.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	_ = session.flow.Close()
	// The session is abandoned; the pending goroutine's release is a no-op.
	s.releaseSignIn(sessionID)
	return nil
}

func (s *Service) CloudflareBuckets(ctx context.Context, sessionID, accountID string) ([]core.Bucket, error) {
	session, err := s.authorizedSession(sessionID)
	if err != nil {
		return nil, err
	}
	if accountID == "" {
		return nil, core.Invalid("Select a Cloudflare account first")
	}
	session.mu.Lock()
	token := session.tokens.AccessToken
	session.mu.Unlock()
	buckets, err := session.client.Buckets(ctx, token, accountID, "default")
	if err != nil {
		return nil, err
	}
	result := make([]core.Bucket, 0, len(buckets))
	for _, bucket := range buckets {
		result = append(result, core.Bucket{Name: bucket.Name})
	}
	return result, nil
}

func (s *Service) authorizedSession(id string) (*signInSession, error) {
	s.signInMu.Lock()
	session, ok := s.signIns[id]
	s.signInMu.Unlock()
	if !ok {
		return nil, core.NewError("not_found", "Start the Cloudflare sign-in again")
	}
	if snapshot := session.snapshot(); snapshot.State != "authorized" {
		return nil, core.NewError("conflict", "The Cloudflare authorization is not complete yet")
	}
	return session, nil
}

// ConnectCloudflare mints a bucket-scoped R2 token from a completed sign-in and
// stores the derived S3 credentials. The R2 token is removed again if the
// connection cannot be saved.
func (s *Service) ConnectCloudflare(ctx context.Context, input CloudflareConnection) (core.Profile, error) {
	session, err := s.authorizedSession(input.SessionID)
	if err != nil {
		return core.Profile{}, err
	}
	accountID, err := pickAccount(session.snapshot().Accounts, input.AccountID)
	if err != nil {
		return core.Profile{}, err
	}
	session.mu.Lock()
	token := session.tokens.AccessToken
	refresh := session.tokens.RefreshToken
	scope := session.tokens.Scope
	session.mu.Unlock()

	p, err := normalizeProfile(core.Profile{
		ID:               input.ProfileID,
		Name:             input.Name,
		ProviderType:     "r2",
		AccountID:        accountID,
		Jurisdiction:     input.Jurisdiction,
		Bucket:           input.Bucket,
		PublicURL:        input.PublicURL,
		ReadOnly:         input.ReadOnly,
		CredentialSource: "cloudflare",
	})
	if err != nil {
		return core.Profile{}, err
	}
	if p.ID == "" {
		p.ID = ID()
	} else if existing, err := s.DB.Profile(ctx, p.ID); err == nil {
		if existing.AccountID != p.AccountID {
			return core.Profile{}, core.Invalid("Sign in with the Cloudflare account used by this connection")
		}
	} else if core.AsError(err).Code != "not_found" {
		return core.Profile{}, err
	}
	minted, err := session.client.CreateR2Token(ctx, token, cloudflare.TokenSpec{
		Name:         tokenName(p.Name, p.Bucket),
		AccountID:    p.AccountID,
		Bucket:       p.Bucket,
		Jurisdiction: p.Jurisdiction,
		ReadOnly:     p.ReadOnly,
	})
	if err != nil {
		return core.Profile{}, err
	}
	creds := cloudflare.S3Credentials(minted)
	if err := s.verifyCredentials(ctx, p, creds); err != nil {
		s.discardToken(ctx, session.client, token, p.AccountID, minted.ID)
		return core.Profile{}, err
	}
	previous := s.authorization(ctx, input.ProfileID)
	record := authorization{
		RefreshToken: refresh,
		AccountID:    p.AccountID,
		TokenID:      minted.ID,
		TokenName:    minted.Name,
		Jurisdiction: p.Jurisdiction,
		ReadOnly:     p.ReadOnly,
		Scope:        scope,
		AuthorizedAt: time.Now().UTC(),
	}
	p, err = s.storeCloudflareProfile(ctx, p, creds, record)
	if err != nil {
		s.discardToken(ctx, session.client, token, p.AccountID, minted.ID)
		return core.Profile{}, err
	}
	if previous.TokenID != "" && previous.TokenID != minted.ID {
		s.discardToken(ctx, session.client, token, previous.AccountID, previous.TokenID)
	}
	s.releaseSignIn(session.id)
	return p, nil
}

func (s *Service) storeCloudflareProfile(ctx context.Context, p core.Profile, creds core.Credentials, record authorization) (core.Profile, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return p, core.NewError("internal", "Cannot encode the Cloudflare authorization")
	}
	err = s.Transfers.WithWriteLock(ctx, func() error {
		now := time.Now().UTC()
		old, err := s.DB.Profile(ctx, p.ID)
		switch {
		case err == nil:
			p.CreatedAt = old.CreatedAt
		case core.AsError(err).Code == "not_found":
			p.CreatedAt = now
		default:
			return err
		}
		p.UpdatedAt = now
		previousCredentials, credentialsErr := s.vault.Get(p.ID)
		previousSecret, secretErr := s.secrets.GetSecret(cloudflareSecret + p.ID)
		restoreCredentials := func() error {
			if credentialsErr != nil {
				return s.vault.Delete(p.ID)
			}
			return s.vault.Set(p.ID, previousCredentials)
		}
		if err := s.vault.Set(p.ID, creds); err != nil {
			return err
		}
		if err := s.secrets.SetSecret(cloudflareSecret+p.ID, string(encoded)); err != nil {
			return errors.Join(err, restoreCredentials())
		}
		if err := s.DB.SaveProfile(ctx, p); err != nil {
			// Keychain writes are outside the database transaction.
			var restoreSecretErr error
			if secretErr != nil {
				restoreSecretErr = s.secrets.DeleteSecret(cloudflareSecret + p.ID)
			} else {
				restoreSecretErr = s.secrets.SetSecret(cloudflareSecret+p.ID, previousSecret)
			}
			return errors.Join(err, restoreSecretErr, restoreCredentials())
		}
		return nil
	})
	return p, err
}

// verifyCredentials proves the minted S3 pair works before anything is stored.
func (s *Service) verifyCredentials(ctx context.Context, p core.Profile, creds core.Credentials) error {
	driver, err := s.factory(ctx, p, creds)
	if err != nil {
		return err
	}
	if p.Bucket != "" {
		_, err = driver.List(ctx, core.ListRequest{ProfileID: p.ID, Bucket: p.Bucket, Limit: 1})
		return err
	}
	lister, ok := driver.(core.BucketLister)
	if !ok {
		return core.NewError("unsupported", "Bucket discovery is unavailable; specify a bucket in the connection")
	}
	_, err = lister.Buckets(ctx)
	return err
}

func (s *Service) discardToken(ctx context.Context, client *cloudflare.Client, accessToken, accountID, tokenID string) {
	if tokenID == "" || accountID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
	defer cancel()
	_ = client.DeleteToken(ctx, accessToken, accountID, tokenID)
}

func (s *Service) authorization(ctx context.Context, profileID string) authorization {
	if profileID == "" {
		return authorization{}
	}
	value, err := s.secrets.GetSecret(cloudflareSecret + profileID)
	if err != nil {
		return authorization{}
	}
	var record authorization
	if json.Unmarshal([]byte(value), &record) != nil {
		return authorization{}
	}
	return record
}

func (s *Service) CloudflareStatus(ctx context.Context, profileID string) (CloudflareStatus, error) {
	record := s.authorization(ctx, profileID)
	return CloudflareStatus{Authorized: record.TokenID != "", TokenName: record.TokenName, AuthorizedAt: record.AuthorizedAt}, nil
}

// ReauthorizeCloudflare mints a replacement token from the stored refresh token
// without opening the browser again.
func (s *Service) ReauthorizeCloudflare(ctx context.Context, profileID string) (core.Profile, error) {
	p, err := s.DB.Profile(ctx, profileID)
	if err != nil {
		return core.Profile{}, err
	}
	if p.CredentialSource != "cloudflare" {
		return core.Profile{}, core.NewError("unsupported", "This connection does not use Cloudflare sign-in")
	}
	return s.remintCloudflare(ctx, p)
}

// Scope changes require a new R2 token; editing the profile cannot alter it.
func (s *Service) remintCloudflare(ctx context.Context, p core.Profile) (core.Profile, error) {
	record := s.authorization(ctx, p.ID)
	if record.RefreshToken == "" {
		return core.Profile{}, core.NewError("credentials_unavailable", "Sign in to Cloudflare again to change this connection")
	}
	client, _, err := s.cloudflareClient(ctx)
	if err != nil {
		return core.Profile{}, err
	}
	tokens, err := client.Refresh(ctx, record.RefreshToken)
	if err != nil {
		return core.Profile{}, err
	}
	minted, err := client.CreateR2Token(ctx, tokens.AccessToken, cloudflare.TokenSpec{
		Name:         tokenName(p.Name, p.Bucket),
		AccountID:    p.AccountID,
		Bucket:       p.Bucket,
		Jurisdiction: p.Jurisdiction,
		ReadOnly:     p.ReadOnly,
	})
	if err != nil {
		return core.Profile{}, err
	}
	creds := cloudflare.S3Credentials(minted)
	if err := s.verifyCredentials(ctx, p, creds); err != nil {
		s.discardToken(ctx, client, tokens.AccessToken, p.AccountID, minted.ID)
		return core.Profile{}, err
	}
	if tokens.RefreshToken != "" {
		record.RefreshToken = tokens.RefreshToken
	}
	previousID, previousAccount := record.TokenID, record.AccountID
	record.AccountID, record.Jurisdiction, record.ReadOnly = p.AccountID, p.Jurisdiction, p.ReadOnly
	record.TokenID, record.TokenName = minted.ID, minted.Name
	record.Scope, record.AuthorizedAt = tokens.Scope, time.Now().UTC()
	p, err = s.storeCloudflareProfile(ctx, p, creds, record)
	if err != nil {
		s.discardToken(ctx, client, tokens.AccessToken, p.AccountID, minted.ID)
		return core.Profile{}, err
	}
	if previousID != "" && previousID != minted.ID {
		s.discardToken(ctx, client, tokens.AccessToken, previousAccount, previousID)
	}
	return p, nil
}

// DisconnectCloudflare revokes the minted R2 token and clears the local
// authorization. The connection stays so it can be authorized again.
func (s *Service) DisconnectCloudflare(ctx context.Context, profileID string, confirmed bool) error {
	if !confirmed {
		return core.NewError("confirmation_required", "Confirm disconnecting this Cloudflare connection")
	}
	p, err := s.DB.Profile(ctx, profileID)
	if err != nil {
		return err
	}
	if p.CredentialSource != "cloudflare" {
		return core.NewError("unsupported", "This connection does not use Cloudflare sign-in")
	}
	record := s.authorization(ctx, profileID)
	client, _, err := s.cloudflareClient(ctx)
	if err != nil {
		return err
	}
	accessToken := ""
	if tokens, refreshErr := client.Refresh(ctx, record.RefreshToken); refreshErr == nil {
		accessToken = tokens.AccessToken
	}
	if accessToken == "" {
		return core.NewError("credentials_unavailable", "Sign in to Cloudflare again before disconnecting this connection")
	}
	if record.TokenID != "" {
		if err := client.DeleteToken(ctx, accessToken, record.AccountID, record.TokenID); err != nil {
			return err
		}
	}
	_ = client.Revoke(ctx, record.RefreshToken)
	return s.Transfers.WithWriteLock(ctx, func() error {
		if err := s.vault.Delete(profileID); err != nil {
			return err
		}
		return s.secrets.DeleteSecret(cloudflareSecret + profileID)
	})
}

func cloudflareScopeChanged(old, next core.Profile) bool {
	if old.Jurisdiction == "" {
		old.Jurisdiction = "default"
	}
	return old.Bucket != next.Bucket || old.AccountID != next.AccountID ||
		old.Jurisdiction != next.Jurisdiction || old.ReadOnly != next.ReadOnly
}

func pickAccount(accounts []CloudflareAccount, id string) (string, error) {
	if len(accounts) == 0 {
		return "", core.NewError("not_found", "The Cloudflare authorization covers no account")
	}
	if id == "" {
		return accounts[0].ID, nil
	}
	for _, account := range accounts {
		if account.ID == id {
			return account.ID, nil
		}
	}
	return "", core.Invalid("Choose one of the authorized Cloudflare accounts")
}

func tokenName(name, bucket string) string {
	value := strings.TrimSpace("Bucky " + name + " " + bucket)
	runes := []rune(value)
	if len(runes) > 100 {
		value = string(runes[:100])
	}
	return value
}

func splitScopes(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	return cleanScopes(parts)
}

func cleanScopes(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] || !validScope(value) {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func validScope(value string) bool {
	if len(value) > 64 {
		return false
	}
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '.' && r != '-' && r != '_' {
			return false
		}
	}
	return true
}
