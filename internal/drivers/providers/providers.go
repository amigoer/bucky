package providers

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/drivers/s3compat"
)

var _ core.Factory = New

func New(ctx context.Context, profile core.Profile, credentials core.Credentials) (core.ObjectStore, error) {
	if profile.ProviderType != "r2" {
		return nil, core.NewError("unsupported", "Only Cloudflare R2 connections are available")
	}
	endpoint, err := r2Endpoint(profile.AccountID, profile.Jurisdiction)
	if err != nil {
		return nil, err
	}
	return s3compat.New(ctx, s3compat.Config{Endpoint: endpoint, Region: "auto", PathStyle: true, Credentials: credentials, ProfileID: profile.ID, ConditionPolicy: s3compat.R2Conditions})
}

func r2Endpoint(accountID, jurisdiction string) (string, error) {
	if len(accountID) != 32 {
		return "", core.Invalid("R2 account ID must contain exactly 32 hexadecimal characters")
	}
	if _, err := hex.DecodeString(accountID); err != nil {
		return "", core.Invalid("R2 account ID must contain exactly 32 hexadecimal characters")
	}
	var suffix string
	switch jurisdiction {
	case "", "default":
		suffix = ".r2.cloudflarestorage.com"
	case "eu", "fedramp":
		suffix = "." + jurisdiction + ".r2.cloudflarestorage.com"
	default:
		return "", core.Invalid("R2 jurisdiction must be default, eu, or fedramp")
	}
	return "https://" + strings.ToLower(accountID) + suffix, nil
}
