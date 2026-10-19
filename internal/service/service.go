package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"maps"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/drivers/cloudflare"
	"github.com/amigoer/bucky/internal/drivers/providers"
	"github.com/amigoer/bucky/internal/profile"
	"github.com/amigoer/bucky/internal/store"
	"github.com/amigoer/bucky/internal/transfer"
)

type Options struct {
	Dir        string
	Vault      profile.Vault
	Secrets    profile.Secrets
	Factory    core.Factory
	Cloudflare func(cloudflare.Config) (*cloudflare.Client, error)
}

type Service struct {
	DB        *store.Store
	Transfers *transfer.Engine

	// OnCloudflareSignIn reports sign-in progress to the entry adapter.
	OnCloudflareSignIn func(CloudflareSignIn)

	vault      profile.Vault
	secrets    profile.Secrets
	factory    core.Factory
	cloudflare func(cloudflare.Config) (*cloudflare.Client, error)

	signInMu sync.Mutex
	signIns  map[string]*signInSession
}

type ProfileInput struct {
	Profile         core.Profile `json:"profile"`
	AccessKeyID     string       `json:"accessKeyId"`
	SecretAccessKey string       `json:"secretAccessKey"`
}

type TransferResult struct {
	Task *core.TransferTask  `json:"task,omitempty"`
	Plan *core.OperationPlan `json:"plan,omitempty"`
}

func New(ctx context.Context, opts Options) (*Service, error) {
	if opts.Dir == "" {
		var err error
		opts.Dir, err = store.DataDir()
		if err != nil {
			return nil, core.NewError("configuration", "Cannot locate the application data directory")
		}
	}
	if opts.Vault == nil {
		opts.Vault = profile.Keychain{}
	}
	if opts.Secrets == nil {
		opts.Secrets = profile.Keychain{}
	}
	if opts.Factory == nil {
		opts.Factory = providers.New
	}
	if opts.Cloudflare == nil {
		opts.Cloudflare = cloudflare.New
	}
	db, err := store.Open(ctx, opts.Dir)
	if err != nil {
		return nil, err
	}
	s := &Service{
		DB:         db,
		vault:      opts.Vault,
		secrets:    opts.Secrets,
		factory:    opts.Factory,
		cloudflare: opts.Cloudflare,
		signIns:    map[string]*signInSession{},
	}
	s.Transfers = transfer.New(db, s.resolve)
	s.Transfers.Writable = s.checkWritable
	return s, nil
}

func (s *Service) Close() error {
	s.signInMu.Lock()
	sessions := make([]*signInSession, 0, len(s.signIns))
	for _, session := range s.signIns {
		sessions = append(sessions, session)
	}
	s.signIns = map[string]*signInSession{}
	s.signInMu.Unlock()
	for _, session := range sessions {
		session.mu.Lock()
		cancel := session.cancel
		session.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		_ = session.flow.Close()
	}
	err := s.Transfers.Close()
	dbErr := s.DB.Close()
	if err != nil {
		return err
	}
	return dbErr
}

func (s *Service) Profiles(ctx context.Context) ([]core.Profile, error) { return s.DB.Profiles(ctx) }

func (s *Service) Tasks(ctx context.Context) ([]core.TransferTask, error) {
	return s.DB.Tasks(ctx)
}

func (s *Service) Plan(ctx context.Context, id string) (core.OperationPlan, error) {
	return s.DB.Plan(ctx, id)
}

func (s *Service) PauseTransfer(ctx context.Context, id string) error {
	return s.Transfers.Pause(ctx, id)
}

func (s *Service) CancelTransfer(ctx context.Context, id string) error {
	return s.Transfers.Cancel(ctx, id)
}

func (s *Service) SaveProfile(ctx context.Context, input ProfileInput) (core.Profile, error) {
	p, err := normalizeProfile(input.Profile)
	if err != nil {
		return p, err
	}
	if p.CredentialSource == "cloudflare" {
		if input.AccessKeyID != "" || input.SecretAccessKey != "" {
			return p, core.Invalid("A Cloudflare sign-in connection stores no S3 keys")
		}
		if p.ID != "" {
			old, err := s.DB.Profile(ctx, p.ID)
			if err != nil {
				return p, err
			}
			if old.CredentialSource != "cloudflare" {
				return p, core.Invalid("Use Cloudflare sign-in to authorize this connection")
			}
			if old.AccountID != p.AccountID {
				return p, core.Invalid("A Cloudflare connection cannot change its account")
			}
			if cloudflareScopeChanged(old, p) {
				return s.remintCloudflare(ctx, p)
			}
		}
	}
	err = s.Transfers.WithWriteLock(ctx, func() error {
		now := time.Now().UTC()
		if p.ID == "" {
			p.ID, p.CreatedAt = ID(), now
		} else {
			old, err := s.DB.Profile(ctx, p.ID)
			if err != nil {
				return err
			}
			p.CreatedAt = old.CreatedAt
		}
		p.UpdatedAt = now
		switch p.CredentialSource {
		case "keychain":
			if input.AccessKeyID != "" || input.SecretAccessKey != "" {
				if input.AccessKeyID == "" || input.SecretAccessKey == "" {
					return core.Invalid("Both R2 S3 access key fields are required")
				}
				if err := s.vault.Set(p.ID, core.Credentials{AccessKeyID: input.AccessKeyID, SecretAccessKey: input.SecretAccessKey}); err != nil {
					return err
				}
			} else if _, err := s.vault.Get(p.ID); err != nil {
				return err
			}
		case "cloudflare":
			if _, err := s.vault.Get(p.ID); err != nil {
				return err
			}
		}
		return s.DB.SaveProfile(ctx, p)
	})
	return p, err
}

// TestDraft checks unsaved connection settings without storing them. Blank
// keychain fields fall back to the credentials saved for an existing profile.
func (s *Service) TestDraft(ctx context.Context, input ProfileInput) ([]core.Bucket, error) {
	p, err := normalizeProfile(input.Profile)
	if err != nil {
		return nil, err
	}
	var creds core.Credentials
	switch {
	case p.CredentialSource == "environment":
		creds, err = profile.Resolve(p, s.vault)
	case input.AccessKeyID != "" || input.SecretAccessKey != "":
		if input.AccessKeyID == "" || input.SecretAccessKey == "" {
			return nil, core.Invalid("Both R2 S3 access key fields are required")
		}
		creds = core.Credentials{AccessKeyID: input.AccessKeyID, SecretAccessKey: input.SecretAccessKey}
	case p.ID != "":
		if _, err = s.DB.Profile(ctx, p.ID); err == nil {
			creds, err = s.vault.Get(p.ID)
		}
	default:
		err = core.Invalid("Both R2 S3 access key fields are required")
	}
	if err != nil {
		return nil, err
	}
	driver, err := s.factory(ctx, p, creds)
	if err != nil {
		return nil, err
	}
	if p.Bucket != "" {
		_, err = driver.List(ctx, core.ListRequest{ProfileID: p.ID, Bucket: p.Bucket, Limit: 1})
		return []core.Bucket{{Name: p.Bucket}}, err
	}
	lister, ok := driver.(core.BucketLister)
	if !ok {
		return nil, core.NewError("unsupported", "Bucket discovery is unavailable; specify a bucket in the connection")
	}
	return lister.Buckets(ctx)
}

func normalizeProfile(p core.Profile) (core.Profile, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len(p.Name) > 100 {
		return p, core.Invalid("A connection name of 1 to 100 characters is required")
	}
	if p.ProviderType == "" {
		p.ProviderType = "r2"
	}
	if p.ProviderType != "r2" {
		return p, core.NewError("unsupported", "Only Cloudflare R2 connections are available")
	}
	if len(p.AccountID) != 32 {
		return p, core.Invalid("The R2 account ID must contain 32 hexadecimal characters")
	}
	if _, err := hex.DecodeString(p.AccountID); err != nil {
		return p, core.Invalid("The R2 account ID must contain 32 hexadecimal characters")
	}
	p.AccountID = strings.ToLower(p.AccountID)
	if p.Jurisdiction == "" {
		p.Jurisdiction = "default"
	}
	if p.Jurisdiction != "default" && p.Jurisdiction != "eu" && p.Jurisdiction != "fedramp" {
		return p, core.Invalid("Unknown R2 jurisdiction")
	}
	if p.Bucket != "" && (strings.ContainsAny(p.Bucket, "/\\\x00\r\n") || len(p.Bucket) > 63) {
		return p, core.Invalid("Invalid bucket name")
	}
	if p.CredentialSource == "" {
		p.CredentialSource = "keychain"
	}
	switch p.CredentialSource {
	case "keychain", "environment":
	case "cloudflare":
		// A Cloudflare sign-in mints a token scoped to a single bucket.
		if p.Bucket == "" {
			return p, core.Invalid("A Cloudflare sign-in connection is scoped to one bucket")
		}
	default:
		return p, core.Invalid("Credentials must come from the system keychain, this process environment or a Cloudflare sign-in")
	}
	if p.PublicURL != "" {
		u, err := url.Parse(p.PublicURL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return p, core.Invalid("The public bucket URL must be an HTTPS URL without credentials, a query or a fragment")
		}
		p.PublicURL = strings.TrimRight(p.PublicURL, "/")
	}
	return p, nil
}

func (s *Service) RemoveProfile(ctx context.Context, id string, confirmed bool) error {
	if !confirmed {
		return core.NewError("confirmation_required", "Confirm removal of the local connection and its saved credentials")
	}
	return s.Transfers.WithWriteLock(ctx, func() error {
		p, err := s.DB.Profile(ctx, id)
		if err != nil {
			return err
		}
		tasks, err := s.DB.Tasks(ctx)
		if err != nil {
			return err
		}
		for _, t := range tasks {
			if t.Object.ProfileID == id && t.Status != "completed" && t.Status != "cancelled" {
				return core.NewError("conflict", "Cancel unfinished transfers before removing this connection")
			}
		}
		if p.CredentialSource == "keychain" || p.CredentialSource == "cloudflare" {
			if err := s.vault.Delete(id); err != nil {
				return err
			}
		}
		if p.CredentialSource == "cloudflare" {
			if err := s.secrets.DeleteSecret(cloudflareSecret + id); err != nil {
				return err
			}
		}
		return s.DB.DeleteProfile(ctx, id)
	})
}

func (s *Service) resolve(ctx context.Context, id string) (core.ObjectStore, error) {
	p, err := s.DB.Profile(ctx, id)
	if err != nil {
		return nil, err
	}
	creds, err := profile.Resolve(p, s.vault)
	if err != nil {
		return nil, err
	}
	return s.factory(ctx, p, creds)
}

func (s *Service) resolveWrite(ctx context.Context, id string) (core.ObjectStore, error) {
	if err := s.checkWritable(ctx, id); err != nil {
		return nil, err
	}
	return s.resolve(ctx, id)
}

func (s *Service) checkWritable(ctx context.Context, id string) error {
	p, err := s.DB.Profile(ctx, id)
	if err != nil {
		return err
	}
	if p.ReadOnly {
		return core.NewError("read_only", "This connection is read-only")
	}
	return nil
}

func (s *Service) validateRef(ctx context.Context, ref core.ObjectRef, object bool) error {
	p, err := s.DB.Profile(ctx, ref.ProfileID)
	if err != nil {
		return err
	}
	if ref.Bucket == "" || strings.ContainsAny(ref.Bucket, "/\\\x00\r\n") {
		return core.Invalid("A valid bucket name is required")
	}
	if p.Bucket != "" && p.Bucket != ref.Bucket {
		return core.NewError("forbidden", "This connection is scoped to a different bucket")
	}
	if object && (ref.Key == "" || len(ref.Key) > 1024 || strings.ContainsRune(ref.Key, '\x00')) {
		return core.Invalid("An object key of 1 to 1024 bytes is required")
	}
	return nil
}

func (s *Service) Buckets(ctx context.Context, id string) ([]core.Bucket, error) {
	p, err := s.DB.Profile(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Bucket != "" {
		return []core.Bucket{{Name: p.Bucket}}, nil
	}
	driver, err := s.resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	lister, ok := driver.(core.BucketLister)
	if !ok {
		return nil, core.NewError("unsupported", "Bucket discovery is unavailable; specify a bucket in the connection")
	}
	return lister.Buckets(ctx)
}

func (s *Service) TestConnection(ctx context.Context, id string) ([]core.Bucket, error) {
	buckets, err := s.Buckets(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := s.DB.Profile(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Bucket != "" {
		_, err = s.List(ctx, core.ListRequest{ProfileID: id, Bucket: p.Bucket, Limit: 1})
	}
	return buckets, err
}

func (s *Service) List(ctx context.Context, req core.ListRequest) (core.ListPage, error) {
	if err := s.validateRef(ctx, core.ObjectRef{ProfileID: req.ProfileID, Bucket: req.Bucket}, false); err != nil {
		return core.ListPage{}, err
	}
	if req.Limit <= 0 || req.Limit > 1000 {
		req.Limit = 200
	}
	if req.Delimiter != "" && req.Delimiter != "/" {
		return core.ListPage{}, core.Invalid("Only the folder delimiter '/' is supported")
	}
	driver, err := s.resolve(ctx, req.ProfileID)
	if err != nil {
		return core.ListPage{}, err
	}
	return driver.List(ctx, req)
}

func (s *Service) Stat(ctx context.Context, ref core.ObjectRef) (core.ObjectInfo, error) {
	if err := s.validateRef(ctx, ref, true); err != nil {
		return core.ObjectInfo{}, err
	}
	driver, err := s.resolve(ctx, ref.ProfileID)
	if err != nil {
		return core.ObjectInfo{}, err
	}
	return driver.Stat(ctx, ref)
}

func (s *Service) ReadText(ctx context.Context, ref core.ObjectRef, limit int64) (core.TextPreview, error) {
	if err := s.validateRef(ctx, ref, true); err != nil {
		return core.TextPreview{}, err
	}
	if limit <= 0 || limit > 1024*1024 {
		limit = 1024 * 1024
	}
	driver, err := s.resolve(ctx, ref.ProfileID)
	if err != nil {
		return core.TextPreview{}, err
	}
	r, err := driver.Read(ctx, ref, core.ReadOptions{Length: limit + 1})
	if err != nil {
		return core.TextPreview{}, err
	}
	defer func() { _ = r.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return core.TextPreview{}, core.AsError(err)
	}
	truncated := int64(len(data)) > limit
	if truncated {
		data = data[:limit]
		for removed := 0; removed < 3 && len(data) > 0 && !utf8.Valid(data); removed++ {
			data = data[:len(data)-1]
		}
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), '\x00') {
		return core.TextPreview{}, core.NewError("unsupported", "This object is not UTF-8 text")
	}
	return core.TextPreview{Text: string(data), ContentType: r.Info.ContentType, Truncated: truncated}, nil
}

func (s *Service) Share(ctx context.Context, ref core.ObjectRef, seconds int64, public bool) (core.ShareLink, error) {
	if err := s.validateRef(ctx, ref, true); err != nil {
		return core.ShareLink{}, err
	}
	if public {
		p, err := s.DB.Profile(ctx, ref.ProfileID)
		if err != nil {
			return core.ShareLink{}, err
		}
		if p.PublicURL == "" {
			return core.ShareLink{}, core.NewError("configuration", "Configure an existing public bucket domain first")
		}
		parts := strings.Split(ref.Key, "/")
		for i := range parts {
			parts[i] = url.PathEscape(parts[i])
		}
		return core.ShareLink{URL: p.PublicURL + "/" + strings.Join(parts, "/"), Public: true, Headers: map[string]string{}}, nil
	}
	if seconds == 0 {
		seconds = 3600
	}
	if seconds < 1 || seconds > 604800 {
		return core.ShareLink{}, core.Invalid("R2 links must expire between 1 second and 7 days")
	}
	driver, err := s.resolve(ctx, ref.ProfileID)
	if err != nil {
		return core.ShareLink{}, err
	}
	presigner, ok := driver.(core.Presigner)
	if !ok {
		return core.ShareLink{}, core.NewError("unsupported", "Presigned links are unavailable")
	}
	return presigner.Presign(ctx, ref, time.Duration(seconds)*time.Second)
}

func (s *Service) Find(ctx context.Context, req core.FindRequest) (core.FindResult, error) {
	result := core.FindResult{Objects: []core.ObjectInfo{}}
	var before time.Time
	if req.Before != "" {
		var err error
		before, err = time.Parse(time.RFC3339, req.Before)
		if err != nil {
			return result, core.Invalid("The before filter must be an RFC3339 timestamp")
		}
	}
	if req.Limit <= 0 || req.Limit > 10000 {
		req.Limit = 1000
	}
	cursor := req.Cursor
	for {
		page, err := s.List(ctx, core.ListRequest{ProfileID: req.ProfileID, Bucket: req.Bucket, Prefix: req.Prefix, Cursor: cursor, Limit: 1000})
		if err != nil {
			return result, err
		}
		for _, obj := range page.Objects {
			result.Scanned++
			if strings.Contains(strings.ToLower(obj.Key), strings.ToLower(req.Contains)) && (before.IsZero() || obj.LastModified.Before(before)) && obj.Size >= req.MinSize && (req.MaxSize <= 0 || obj.Size <= req.MaxSize) {
				result.Objects = append(result.Objects, obj)
			}
		}
		result.NextCursor = page.NextCursor
		if page.NextCursor == "" {
			result.Complete = true
			return result, nil
		}
		if len(result.Objects) >= req.Limit {
			return result, nil
		}
		if page.NextCursor == cursor {
			return result, core.NewError("protocol", "The storage service repeated a pagination cursor")
		}
		cursor = page.NextCursor
	}
}

func (s *Service) Usage(ctx context.Context, id, bucket, prefix string) (core.Usage, error) {
	return s.ScanUsage(ctx, id, bucket, prefix, nil)
}

// ScanUsage reports the running totals after each page. On error the partial
// totals are returned with Complete unset; only complete scans are cached.
func (s *Service) ScanUsage(ctx context.Context, id, bucket, prefix string, progress func(core.Usage)) (core.Usage, error) {
	u := core.Usage{ProfileID: id, Bucket: bucket, Prefix: prefix, Types: map[string]int64{}, Counts: map[string]int64{}, ScannedAt: time.Now().UTC()}
	cursor := ""
	for {
		page, err := s.List(ctx, core.ListRequest{ProfileID: id, Bucket: bucket, Prefix: prefix, Cursor: cursor, Limit: 1000})
		if err != nil {
			u.ScannedAt = time.Now().UTC()
			return u, err
		}
		for _, obj := range page.Objects {
			u.Objects++
			u.Bytes += obj.Size
			ext := strings.ToLower(path.Ext(obj.Key))
			if ext == "" {
				ext = "(none)"
			}
			u.Types[ext] += obj.Size
			u.Counts[ext]++
		}
		if page.NextCursor == "" {
			u.Complete = true
			u.ScannedAt = time.Now().UTC()
			return u, s.DB.SaveUsage(ctx, u)
		}
		if page.NextCursor == cursor {
			u.ScannedAt = time.Now().UTC()
			return u, core.NewError("protocol", "The storage service repeated a pagination cursor")
		}
		cursor = page.NextCursor
		if progress != nil {
			progress(cloneUsage(u))
		}
	}
}

func cloneUsage(u core.Usage) core.Usage {
	u.Types = maps.Clone(u.Types)
	u.Counts = maps.Clone(u.Counts)
	return u
}

func ID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic("secure random source unavailable")
	}
	return hex.EncodeToString(value[:])
}

func localFile(filename string) (os.FileInfo, error) {
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() {
		return nil, core.Invalid("Select an existing regular file")
	}
	return info, nil
}
