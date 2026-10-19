package profile

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/amigoer/bucky/internal/core"
	"github.com/zalando/go-keyring"
)

type Vault interface {
	Get(string) (core.Credentials, error)
	Set(string, core.Credentials) error
	Delete(string) error
}

// Secrets keeps single opaque values, such as a Cloudflare refresh token, next
// to the credentials they belong to.
type Secrets interface {
	GetSecret(string) (string, error)
	SetSecret(string, string) error
	DeleteSecret(string) error
}

type Keychain struct{}

func (Keychain) Get(id string) (core.Credentials, error) {
	value, err := keyring.Get("Bucky", id)
	if err != nil {
		return core.Credentials{}, core.NewError("credentials_unavailable", "Cannot read credentials from the system keychain")
	}
	var creds core.Credentials
	if err := json.Unmarshal([]byte(value), &creds); err != nil {
		return creds, core.NewError("credentials_unavailable", "The saved credentials are invalid; enter them again")
	}
	return creds, nil
}

func (Keychain) Set(id string, creds core.Credentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return core.NewError("credentials_unavailable", "Cannot encode credentials")
	}
	if err := keyring.Set("Bucky", id, string(data)); err != nil {
		return core.NewError("credentials_unavailable", "Cannot save credentials to the system keychain")
	}
	return nil
}

func (Keychain) Delete(id string) error {
	if err := keyring.Delete("Bucky", id); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return core.NewError("credentials_unavailable", "Cannot remove credentials from the system keychain")
	}
	return nil
}

func (Keychain) GetSecret(key string) (string, error) {
	value, err := keyring.Get("Bucky", key)
	if err != nil {
		return "", core.NewError("credentials_unavailable", "Cannot read the saved Cloudflare authorization")
	}
	return value, nil
}

func (Keychain) SetSecret(key, value string) error {
	if err := keyring.Set("Bucky", key, value); err != nil {
		return core.NewError("credentials_unavailable", "Cannot save the Cloudflare authorization to the system keychain")
	}
	return nil
}

func (Keychain) DeleteSecret(key string) error {
	if err := keyring.Delete("Bucky", key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return core.NewError("credentials_unavailable", "Cannot remove the Cloudflare authorization from the system keychain")
	}
	return nil
}

func Resolve(p core.Profile, vault Vault) (core.Credentials, error) {
	if p.CredentialSource == "environment" {
		c := core.Credentials{
			AccessKeyID: os.Getenv("BUCKY_R2_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("BUCKY_R2_SECRET_ACCESS_KEY"),
			SessionToken: os.Getenv("BUCKY_R2_SESSION_TOKEN"),
		}
		if c.AccessKeyID == "" || c.SecretAccessKey == "" {
			return c, core.NewError("credentials_unavailable", "Set BUCKY_R2_ACCESS_KEY_ID and BUCKY_R2_SECRET_ACCESS_KEY for this process")
		}
		return c, nil
	}
	return vault.Get(p.ID)
}
