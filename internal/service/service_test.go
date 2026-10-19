package service

import (
	"context"
	"io"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amigoer/bucky/internal/core"
)

type memoryVault struct {
	mu    sync.Mutex
	creds map[string]core.Credentials
}

func (v *memoryVault) Get(id string) (core.Credentials, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	c, ok := v.creds[id]
	if !ok {
		return c, core.NewError("credentials_unavailable", "missing")
	}
	return c, nil
}

func (v *memoryVault) Set(id string, c core.Credentials) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.creds[id] = c
	return nil
}

func (v *memoryVault) Delete(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.creds, id)
	return nil
}

// memoryStore pages after two objects so tests exercise pagination.
type memoryStore struct {
	mu      sync.Mutex
	objects map[string]core.ObjectInfo
	lists   int
}

func (m *memoryStore) List(_ context.Context, req core.ListRequest) (core.ListPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lists++
	keys := []string{}
	for key := range m.objects {
		if strings.HasPrefix(key, req.Prefix) && key > req.Cursor {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	page := core.ListPage{Objects: []core.ObjectInfo{}, Prefixes: []string{}}
	limit := min(int(req.Limit), 2)
	for i, key := range keys {
		if i == limit {
			page.NextCursor = keys[i-1]
			break
		}
		page.Objects = append(page.Objects, m.objects[key])
	}
	return page, nil
}

func (m *memoryStore) Stat(_ context.Context, ref core.ObjectRef) (core.ObjectInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	info, ok := m.objects[ref.Key]
	if !ok {
		return info, core.NewError("not_found", "missing")
	}
	info.ObjectRef = ref
	return info, nil
}

func (m *memoryStore) Read(context.Context, core.ObjectRef, core.ReadOptions) (*core.ObjectReader, error) {
	return nil, core.NewError("unsupported", "not implemented")
}

func (m *memoryStore) Put(context.Context, core.ObjectRef, io.Reader, int64, core.PutOptions) (core.ObjectInfo, error) {
	return core.ObjectInfo{}, core.NewError("unsupported", "not implemented")
}

func (m *memoryStore) Delete(context.Context, []core.ObjectInfo) ([]core.ItemResult, error) {
	return nil, core.NewError("unsupported", "not implemented")
}

func (m *memoryStore) Capabilities() core.Capabilities { return core.Capabilities{} }

func (m *memoryStore) Buckets(context.Context) ([]core.Bucket, error) {
	return []core.Bucket{{Name: "media"}, {Name: "assets"}}, nil
}

type fixture struct {
	svc   *Service
	store *memoryStore
	vault *memoryVault
	creds []core.Credentials
	ref   func(string) core.ObjectRef
}

func newFixture(t *testing.T, keys ...string) *fixture {
	t.Helper()
	f := &fixture{store: &memoryStore{objects: map[string]core.ObjectInfo{}}, vault: &memoryVault{creds: map[string]core.Credentials{}}}
	for i, key := range keys {
		f.store.objects[key] = core.ObjectInfo{ObjectRef: core.ObjectRef{ProfileID: "p1", Bucket: "media", Key: key}, Size: int64(10 * (i + 1)), ETag: "e" + key, LastModified: time.Unix(1700000000, 0)}
	}
	svc, err := New(context.Background(), Options{Dir: t.TempDir(), Vault: f.vault, Factory: func(_ context.Context, _ core.Profile, c core.Credentials) (core.ObjectStore, error) {
		f.creds = append(f.creds, c)
		return f.store, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	f.svc = svc
	f.vault.creds["p1"] = core.Credentials{AccessKeyID: "saved", SecretAccessKey: "saved-secret"}
	if err := svc.DB.SaveProfile(context.Background(), core.Profile{ID: "p1", Name: "Test", ProviderType: "r2", AccountID: strings.Repeat("a", 32), CredentialSource: "keychain"}); err != nil {
		t.Fatal(err)
	}
	f.ref = func(key string) core.ObjectRef { return core.ObjectRef{ProfileID: "p1", Bucket: "media", Key: key} }
	return f
}

func TestCreatePlanKeepsPathsBelowBase(t *testing.T) {
	f := newFixture(t, "photos/2024/a.jpg", "photos/2024/sub/b.jpg", "photos/c.jpg")
	plan, err := f.svc.CreatePlan(context.Background(), core.PlanRequest{
		Operation: "copy", Base: "photos/", Target: f.ref("archive/"),
		Sources: []core.ObjectRef{f.ref("photos/2024/a.jpg"), f.ref("photos/2024/sub/b.jpg"), f.ref("photos/c.jpg")},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, item := range plan.Items {
		got = append(got, item.Target.Key)
	}
	want := []string{"archive/2024/a.jpg", "archive/2024/sub/b.jpg", "archive/c.jpg"}
	if !slices.Equal(got, want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	if plan.Bytes != 60 {
		t.Fatalf("bytes = %d", plan.Bytes)
	}
}

func TestCreatePlanRejectsInvalidBase(t *testing.T) {
	f := newFixture(t, "photos/a.jpg", "other/b.jpg")
	cases := []core.PlanRequest{
		{Operation: "move", Base: "photos", Target: f.ref("archive/"), Sources: []core.ObjectRef{f.ref("photos/a.jpg")}},
		{Operation: "move", Base: "photos/", Target: f.ref("archive/"), Sources: []core.ObjectRef{f.ref("other/b.jpg")}},
		{Operation: "move", Base: "photos/", Target: f.ref("archive"), Sources: []core.ObjectRef{f.ref("photos/a.jpg")}},
	}
	for i, req := range cases {
		if _, err := f.svc.CreatePlan(context.Background(), req); core.AsError(err).Code != "invalid_argument" {
			t.Fatalf("case %d: err = %v", i, err)
		}
	}
}

func TestScanUsageReportsProgressAndCounts(t *testing.T) {
	f := newFixture(t, "a.png", "b.png", "c.txt", "d", "e.PNG")
	var progress []int64
	u, err := f.svc.ScanUsage(context.Background(), "p1", "media", "", func(u core.Usage) {
		progress = append(progress, u.Objects)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !u.Complete || u.Objects != 5 || u.Bytes != 150 {
		t.Fatalf("usage = %+v", u)
	}
	if u.Counts[".png"] != 3 || u.Counts[".txt"] != 1 || u.Counts["(none)"] != 1 {
		t.Fatalf("counts = %v", u.Counts)
	}
	if u.Types[".png"] != 10+20+50 {
		t.Fatalf("types = %v", u.Types)
	}
	if !slices.Equal(progress, []int64{2, 4}) {
		t.Fatalf("progress = %v", progress)
	}
}

func TestScanUsageReturnsPartialTotalsWhenCancelled(t *testing.T) {
	f := newFixture(t, "a.png", "b.png", "c.txt", "d.txt")
	ctx, cancel := context.WithCancel(context.Background())
	u, err := f.svc.ScanUsage(ctx, "p1", "media", "", func(core.Usage) { cancel() })
	if err == nil || u.Complete || u.Objects != 2 {
		t.Fatalf("usage = %+v, err = %v", u, err)
	}
}

func TestTestDraftUsesEnteredOrSavedCredentials(t *testing.T) {
	f := newFixture(t)
	draft := core.Profile{Name: "Draft", AccountID: strings.Repeat("b", 32)}
	buckets, err := f.svc.TestDraft(context.Background(), ProfileInput{Profile: draft, AccessKeyID: "new", SecretAccessKey: "new-secret"})
	if err != nil || len(buckets) != 2 {
		t.Fatalf("buckets = %v, err = %v", buckets, err)
	}
	if f.creds[len(f.creds)-1].AccessKeyID != "new" {
		t.Fatalf("entered credentials were not used")
	}
	if _, err := f.svc.TestDraft(context.Background(), ProfileInput{Profile: draft, AccessKeyID: "only"}); core.AsError(err).Code != "invalid_argument" {
		t.Fatalf("partial keys: err = %v", err)
	}
	if _, err := f.svc.TestDraft(context.Background(), ProfileInput{Profile: draft}); core.AsError(err).Code != "invalid_argument" {
		t.Fatalf("missing keys: err = %v", err)
	}
	saved := draft
	saved.ID, saved.Bucket = "p1", "media"
	buckets, err = f.svc.TestDraft(context.Background(), ProfileInput{Profile: saved})
	if err != nil || len(buckets) != 1 || buckets[0].Name != "media" {
		t.Fatalf("buckets = %v, err = %v", buckets, err)
	}
	if f.creds[len(f.creds)-1].AccessKeyID != "saved" || f.store.lists == 0 {
		t.Fatalf("saved credentials or bucket listing were not used")
	}
	t.Setenv("BUCKY_R2_ACCESS_KEY_ID", "env")
	t.Setenv("BUCKY_R2_SECRET_ACCESS_KEY", "env-secret")
	env := draft
	env.CredentialSource = "environment"
	if _, err := f.svc.TestDraft(context.Background(), ProfileInput{Profile: env}); err != nil {
		t.Fatal(err)
	}
	if f.creds[len(f.creds)-1].AccessKeyID != "env" {
		t.Fatalf("environment credentials were not used")
	}
	profiles, err := f.svc.Profiles(context.Background())
	if err != nil || len(profiles) != 1 {
		t.Fatalf("a draft test saved a profile: %d", len(profiles))
	}
}
