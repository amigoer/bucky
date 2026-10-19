package cloudflare

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"

	"github.com/amigoer/bucky/internal/core"
)

// R2 exposes bucket-scoped S3 permissions under these permission group names.
// The IDs are resolved at runtime because they are not stable across accounts.
const (
	PermissionGroupWrite = "Workers R2 Storage Bucket Item Write"
	PermissionGroupRead  = "Workers R2 Storage Bucket Item Read"

	bucketResourceScope = "com.cloudflare.edge.r2.bucket"
	jurisdictionHeader  = "cf-r2-jurisdiction"
)

var bucketName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Bucket struct {
	Name         string `json:"name"`
	CreationDate string `json:"creation_date"`
	Jurisdiction string `json:"jurisdiction"`
}

type PermissionGroup struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

type Token struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TokenSpec struct {
	Name         string
	AccountID    string
	Bucket       string
	Jurisdiction string
	ReadOnly     bool
}

func (c *Client) Accounts(ctx context.Context, token string) ([]Account, error) {
	var result []Account
	if err := c.do(ctx, token, "GET", "/accounts", nil, &result, nil); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Buckets(ctx context.Context, token, accountID, jurisdiction string) ([]Bucket, error) {
	var result struct {
		Buckets []Bucket `json:"buckets"`
	}
	headers := map[string]string{}
	if jurisdiction != "" && jurisdiction != "default" {
		headers[jurisdictionHeader] = jurisdiction
	}
	if err := c.do(ctx, token, "GET", "/accounts/"+url.PathEscape(accountID)+"/r2/buckets", nil, &result, headers); err != nil {
		return nil, err
	}
	return result.Buckets, nil
}

func (c *Client) PermissionGroups(ctx context.Context, token, accountID string) ([]PermissionGroup, error) {
	query := url.Values{"scope": {bucketResourceScope}}
	var result []PermissionGroup
	path := "/accounts/" + url.PathEscape(accountID) + "/tokens/permission_groups?" + query.Encode()
	if err := c.do(ctx, token, "GET", path, nil, &result, nil); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateR2Token mints a token scoped to one bucket. Its id and the SHA-256 of
// its value are the S3 credentials for that bucket.
func (c *Client) CreateR2Token(ctx context.Context, token string, spec TokenSpec) (Token, error) {
	if !bucketName.MatchString(spec.Bucket) {
		return Token{}, core.Invalid("The bucket name cannot be used in a bucket-scoped R2 token")
	}
	groups, err := c.PermissionGroups(ctx, token, spec.AccountID)
	if err != nil {
		return Token{}, err
	}
	wanted := PermissionGroupWrite
	if spec.ReadOnly {
		wanted = PermissionGroupRead
	}
	group := ""
	for _, candidate := range groups {
		if candidate.Name == wanted {
			group = candidate.ID
			break
		}
	}
	if group == "" {
		return Token{}, core.NewError("configuration", "Cloudflare did not offer the "+wanted+" permission; authorize with the R2 scopes enabled")
	}
	jurisdiction := spec.Jurisdiction
	if jurisdiction == "" {
		jurisdiction = "default"
	}
	resource := bucketResourceScope + "." + spec.AccountID + "_" + jurisdiction + "_" + spec.Bucket
	body := map[string]any{
		"name": spec.Name,
		"policies": []map[string]any{{
			"effect":            "allow",
			"resources":         map[string]string{resource: "*"},
			"permission_groups": []map[string]string{{"id": group}},
		}},
	}
	var result Token
	if err := c.do(ctx, token, "POST", "/accounts/"+url.PathEscape(spec.AccountID)+"/tokens", body, &result, nil); err != nil {
		return Token{}, err
	}
	if result.ID == "" || result.Value == "" {
		return Token{}, core.NewError("provider_error", "Cloudflare created no usable R2 token")
	}
	return result, nil
}

func (c *Client) DeleteToken(ctx context.Context, token, accountID, tokenID string) error {
	path := "/accounts/" + url.PathEscape(accountID) + "/tokens/" + url.PathEscape(tokenID)
	return c.do(ctx, token, "DELETE", path, nil, nil, nil)
}

// S3Credentials derives the S3 key pair of an R2 token. The secret is never
// shown by the dashboard, so it is computed here from the token value.
func S3Credentials(token Token) core.Credentials {
	sum := sha256.Sum256([]byte(token.Value))
	return core.Credentials{AccessKeyID: token.ID, SecretAccessKey: hex.EncodeToString(sum[:])}
}
