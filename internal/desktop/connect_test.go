package desktop

import (
	"testing"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
)

func TestSetAuthMethodNewConnection(t *testing.T) {
	app := newTestApp(t)
	app.openConnect("")
	if !app.canSwitchAuth() {
		t.Fatal("a new connection should offer both methods")
	}

	app.setAuthMethod(true)
	if got := app.connect.profile.CredentialSource; got != "cloudflare" {
		t.Fatalf("credential source = %q, want cloudflare", got)
	}

	app.connect.signInBusy = true
	app.connect.accounts = []service.CloudflareAccount{{ID: "acc1", Name: "Acme"}}
	app.connect.buckets = []core.Bucket{{Name: "assets"}}
	app.connect.accountID2, app.connect.bucket2 = "acc1", "assets"
	app.setAuthMethod(false)
	state := app.connect
	if state.profile.CredentialSource != "keychain" {
		t.Fatalf("credential source = %q, want keychain", state.profile.CredentialSource)
	}
	if state.signInBusy || state.accounts != nil || state.buckets != nil || state.accountID2 != "" || state.bucket2 != "" {
		t.Fatalf("leaving the sign-in kept its state: %+v", state)
	}
}

func TestSetAuthMethodKeepsEnvironmentSource(t *testing.T) {
	app := newTestApp(t)
	app.openConnect("")
	app.connect.credentialSource = "environment"
	app.setAuthMethod(true)
	app.setAuthMethod(false)
	if got := app.connect.profile.CredentialSource; got != "environment" {
		t.Fatalf("credential source = %q, want environment", got)
	}
}

func TestSavedCloudflareConnectionKeepsSignIn(t *testing.T) {
	app := newTestApp(t)
	withProfile(app)
	app.profiles[0].CredentialSource = "cloudflare"
	app.connect = connectState{open: true, editing: "p1", profile: app.profiles[0]}
	if app.canSwitchAuth() {
		t.Fatal("a saved Cloudflare connection should not offer access keys")
	}

	app.profiles[0].CredentialSource = "keychain"
	if !app.canSwitchAuth() {
		t.Fatal("a saved key connection should offer the Cloudflare sign-in")
	}
}
