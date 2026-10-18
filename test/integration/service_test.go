//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
)

func TestServiceWorkflows(t *testing.T) {
	for _, backend := range localBackends() {
		t.Run(backend.name, func(t *testing.T) {
			f := startFixture(t, backend)
			svc, profile := startService(t, f)
			t.Run("transfer", func(t *testing.T) { testServiceTransfer(t, f, svc, profile) })
			t.Run("plans", func(t *testing.T) { testServicePlans(t, f, svc, profile) })
			t.Run("read only", func(t *testing.T) { testServiceReadOnly(t, f, svc, profile) })
		})
	}
}

func testServiceTransfer(t *testing.T, f *fixture, svc *service.Service, profile core.Profile) {
	dir := t.TempDir()
	small := []byte("small content")
	smallPath := writeFile(t, dir, "small.txt", small)
	smallRef := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/transfer/small.txt"}

	result, err := svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: smallRef, LocalPath: smallPath})
	requireNoError(t, "start small upload", err)
	if result.Task == nil || result.Plan != nil {
		t.Fatal("A new object upload must start a transfer without a plan")
	}
	task := waitTask(t, svc, result.Task.ID)
	if task.Status != "completed" || task.Completed != int64(len(small)) || task.Verification == "" {
		t.Fatalf("Small upload did not complete with verification")
	}
	stat, err := svc.Stat(t.Context(), smallRef)
	requireNoError(t, "stat uploaded object", err)
	if stat.Size != int64(len(small)) {
		t.Fatalf("Uploaded object size mismatch")
	}

	share, err := svc.Share(t.Context(), smallRef, 3600, false)
	requireNoError(t, "share uploaded object", err)
	status, body := getPresigned(t, t.Context(), share, "", 1<<20)
	if status != 200 || !bytes.Equal(body, small) {
		t.Fatalf("Shared object returned status %d and different content", status)
	}

	downloadDir := filepath.Join(dir, "downloads")
	requireNoError(t, "create download directory", os.MkdirAll(downloadDir, 0o700))
	smallDownload := filepath.Join(downloadDir, "small.txt")
	result, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "download", Object: smallRef, LocalPath: smallDownload})
	requireNoError(t, "start small download", err)
	if result.Task == nil {
		t.Fatal("Download did not start a transfer")
	}
	waitTask(t, svc, result.Task.ID)
	if got := readFile(t, smallDownload); !bytes.Equal(got, small) {
		t.Fatal("Downloaded content does not match the upload")
	}

	planContent := []byte("plan upload v1")
	planPath := writeFile(t, dir, "plan.txt", planContent)
	planRef := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/transfer/plan.txt"}
	result, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: planRef, LocalPath: planPath})
	requireNoError(t, "start plan seed upload", err)
	waitTask(t, svc, result.Task.ID)

	updated := []byte("plan upload v2 with new bytes")
	writeFile(t, dir, "plan.txt", updated)
	result, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: planRef, LocalPath: planPath})
	requireNoError(t, "plan existing upload", err)
	if result.Plan == nil || result.Task != nil {
		t.Fatal("Uploading over an existing object must produce a plan")
	}
	plan := *result.Plan
	if plan.Operation != "upload" || len(plan.Items) != 1 || plan.Items[0].Existing == nil || plan.Bytes != int64(len(updated)) {
		t.Fatalf("Upload plan does not describe the overwrite")
	}
	_, err = svc.ExecutePlan(t.Context(), plan.ID, false)
	requireErrorCode(t, err, "confirmation_required")
	executed, err := svc.ExecutePlan(t.Context(), plan.ID, true)
	requireNoError(t, "execute confirmed upload plan", err)
	if len(executed.Results) != 1 || !strings.HasPrefix(executed.Results[0].Status, "task:") {
		t.Fatalf("Upload plan did not queue a transfer")
	}
	waitTask(t, svc, strings.TrimPrefix(executed.Results[0].Status, "task:"))
	preview, err := svc.ReadText(t.Context(), planRef, 1<<20)
	requireNoError(t, "read overwritten object", err)
	if preview.Text != string(updated) {
		t.Fatal("Confirmed overwrite did not replace the object content")
	}

	large := makeMultipartData(17 << 20)
	largePath := filepath.Join(dir, "large.bin")
	requireNoError(t, "write large source", os.WriteFile(largePath, large, 0o600))
	largeRef := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/transfer/large.bin"}
	result, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: largeRef, LocalPath: largePath})
	requireNoError(t, "start multipart upload", err)
	task = waitTask(t, svc, result.Task.ID)
	if task.Size != int64(len(large)) || task.Completed != int64(len(large)) {
		t.Fatalf("Multipart upload did not report the full size")
	}
	largeDownload := filepath.Join(downloadDir, "large.bin")
	result, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "download", Object: largeRef, LocalPath: largeDownload})
	requireNoError(t, "start multipart download", err)
	waitTask(t, svc, result.Task.ID)
	if got := sha256File(t, largeDownload); got != sha256Bytes(large) {
		t.Fatal("Large download does not match the source")
	}

	t.Run("download resume", func(t *testing.T) { testServiceDownloadResume(t, f, svc, profile) })
	t.Run("upload resume", func(t *testing.T) { testServiceUploadResume(t, f, svc, profile) })

	found, err := svc.Find(t.Context(), core.FindRequest{ProfileID: profile.ID, Bucket: f.bucket, Prefix: "svc/transfer/", Contains: "plan", Limit: 100})
	requireNoError(t, "find objects", err)
	if !found.Complete || len(found.Objects) != 1 || found.Objects[0].Key != planRef.Key {
		t.Fatalf("Find did not return the expected object")
	}
	usage, err := svc.Usage(t.Context(), profile.ID, f.bucket, "svc/transfer/")
	requireNoError(t, "scan usage", err)
	expectedBytes := int64(len(small) + len(updated) + len(large))
	if !usage.Complete || usage.Objects != 3 || usage.Bytes != expectedBytes || usage.Types[".txt"] != int64(len(small)+len(updated)) || usage.Types[".bin"] != int64(len(large)) {
		t.Fatalf("Usage did not match the uploaded objects")
	}
}

func testServiceDownloadResume(t *testing.T, f *fixture, svc *service.Service, profile core.Profile) {
	dir := t.TempDir()
	data := makeMultipartData(2 << 20)
	source := filepath.Join(dir, "source.bin")
	requireNoError(t, "write resume source", os.WriteFile(source, data, 0o600))
	ref := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/resume/download.bin"}
	result, err := svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: ref, LocalPath: source})
	requireNoError(t, "seed download resume object", err)
	waitTask(t, svc, result.Task.ID)

	svc.Transfers.SetRate(256 << 10)
	t.Cleanup(func() { svc.Transfers.SetRate(0) })
	downloads := filepath.Join(dir, "downloads")
	requireNoError(t, "create resume download directory", os.MkdirAll(downloads, 0o700))
	destination := filepath.Join(downloads, "download.bin")
	result, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "download", Object: ref, LocalPath: destination})
	requireNoError(t, "start slow download", err)
	waitTaskProgress(t, svc, result.Task.ID)
	requireNoError(t, "pause download", svc.PauseTransfer(t.Context(), result.Task.ID))
	paused := taskByID(t, svc, result.Task.ID)
	if paused.Status != "paused" || paused.Completed <= 0 || paused.RemoteETag == "" {
		t.Fatalf("Download did not pause with resumable state")
	}
	svc.Transfers.SetRate(0)
	_, err = svc.ResumeTransfer(t.Context(), result.Task.ID)
	requireNoError(t, "resume download", err)
	waitTask(t, svc, result.Task.ID)
	if got := sha256File(t, destination); got != sha256Bytes(data) {
		t.Fatal("Resumed download does not match the source")
	}
}

func testServiceUploadResume(t *testing.T, f *fixture, svc *service.Service, profile core.Profile) {
	dir := t.TempDir()
	data := makeMultipartData(18 << 20)
	source := filepath.Join(dir, "source.bin")
	requireNoError(t, "write resume upload source", os.WriteFile(source, data, 0o600))
	ref := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/resume/upload.bin"}

	svc.Transfers.SetRate(256 << 10)
	t.Cleanup(func() { svc.Transfers.SetRate(0) })
	result, err := svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: ref, LocalPath: source})
	requireNoError(t, "start slow upload", err)
	waitTaskProgress(t, svc, result.Task.ID)
	requireNoError(t, "pause upload", svc.PauseTransfer(t.Context(), result.Task.ID))
	paused := taskByID(t, svc, result.Task.ID)
	if paused.Status != "paused" || paused.Completed <= 0 || paused.UploadID == "" {
		t.Fatalf("Upload did not pause with resumable multipart state")
	}
	svc.Transfers.SetRate(0)
	_, err = svc.ResumeTransfer(t.Context(), result.Task.ID)
	requireNoError(t, "resume upload", err)
	completed := waitTask(t, svc, result.Task.ID)
	if completed.Size != int64(len(data)) || completed.Completed != int64(len(data)) {
		t.Fatalf("Resumed upload did not report the full size")
	}
	stat, err := svc.Stat(t.Context(), ref)
	requireNoError(t, "stat resumed upload", err)
	if stat.Size != int64(len(data)) {
		t.Fatalf("Resumed upload has the wrong size")
	}
	downloaded := filepath.Join(dir, "verify.bin")
	result, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "download", Object: ref, LocalPath: downloaded})
	requireNoError(t, "verify resumed upload", err)
	waitTask(t, svc, result.Task.ID)
	if got := sha256File(t, downloaded); got != sha256Bytes(data) {
		t.Fatal("Resumed upload content does not match the source")
	}
}

func taskByID(t *testing.T, svc *service.Service, id string) core.TransferTask {
	t.Helper()
	tasks, err := svc.Tasks(t.Context())
	requireNoError(t, "list transfer tasks", err)
	for _, task := range tasks {
		if task.ID == id {
			return task
		}
	}
	t.Fatal("Transfer task was not found")
	return core.TransferTask{}
}

func waitTaskProgress(t *testing.T, svc *service.Service, id string) core.TransferTask {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	for {
		task := taskByID(t, svc, id)
		switch task.Status {
		case "completed":
			t.Fatal("Transfer finished before it could be paused")
		case "failed", "cancelled":
			code := "unknown"
			if task.Error != nil {
				code = task.Error.Code
			}
			t.Fatalf("Transfer ended with status %s and code %s", task.Status, code)
		}
		if task.Completed > 0 {
			return task
		}
		select {
		case <-ctx.Done():
			t.Fatal("Transfer did not make progress before the test deadline")
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func testServicePlans(t *testing.T, f *fixture, svc *service.Service, profile core.Profile) {
	dir := t.TempDir()
	basePath := writeFile(t, dir, "base.txt", []byte("base content"))
	baseRef := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/plans/base.txt"}
	result, err := svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: baseRef, LocalPath: basePath})
	requireNoError(t, "start base upload", err)
	waitTask(t, svc, result.Task.ID)

	copiedRef := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/plans/copied.txt"}
	plan, err := svc.CreatePlan(t.Context(), core.PlanRequest{Operation: "copy", Sources: []core.ObjectRef{baseRef}, Target: copiedRef})
	requireNoError(t, "create copy plan", err)
	if plan.Items[0].Existing != nil {
		t.Fatal("Copy plan unexpectedly found an existing destination")
	}
	_, err = svc.ExecutePlan(t.Context(), plan.ID, false)
	requireErrorCode(t, err, "confirmation_required")
	executed, err := svc.ExecutePlan(t.Context(), plan.ID, true)
	requireNoError(t, "execute copy plan", err)
	if executed.Status != "completed" || len(executed.Results) != 1 || executed.Results[0].Error != nil {
		t.Fatalf("Copy plan did not complete cleanly")
	}
	if _, err := svc.Stat(t.Context(), copiedRef); err != nil {
		t.Fatal("Copy plan did not create the destination")
	}
	if _, err := svc.Stat(t.Context(), baseRef); err != nil {
		t.Fatal("Copy plan removed the source")
	}

	movedRef := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/plans/moved.txt"}
	movePlan, err := svc.CreatePlan(t.Context(), core.PlanRequest{Operation: "move", Sources: []core.ObjectRef{copiedRef}, Target: movedRef})
	requireNoError(t, "create move plan", err)
	moveExecuted, err := svc.ExecutePlan(t.Context(), movePlan.ID, true)
	requireNoError(t, "execute move plan", err)
	if moveExecuted.Status != "completed" || len(moveExecuted.Results) != 1 || moveExecuted.Results[0].Error != nil {
		t.Fatalf("Move plan did not complete cleanly")
	}
	if _, err := svc.Stat(t.Context(), movedRef); err != nil {
		t.Fatal("Move plan did not create the destination")
	}
	_, err = svc.Stat(t.Context(), copiedRef)
	requireErrorCode(t, err, "not_found")

	deletePlan, err := svc.CreatePlan(t.Context(), core.PlanRequest{Operation: "delete", Sources: []core.ObjectRef{movedRef}})
	requireNoError(t, "create delete plan", err)
	_, err = svc.ExecutePlan(t.Context(), deletePlan.ID, false)
	requireErrorCode(t, err, "confirmation_required")
	deleteExecuted, err := svc.ExecutePlan(t.Context(), deletePlan.ID, true)
	requireNoError(t, "execute delete plan", err)
	if deleteExecuted.Status != "completed" || len(deleteExecuted.Results) != 1 || deleteExecuted.Results[0].Error != nil {
		t.Fatalf("Delete plan did not complete cleanly")
	}
	_, err = svc.Stat(t.Context(), movedRef)
	requireErrorCode(t, err, "not_found")

	stalePlan, err := svc.CreatePlan(t.Context(), core.PlanRequest{Operation: "copy", Sources: []core.ObjectRef{baseRef}, Target: core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/plans/stale.txt"}})
	requireNoError(t, "create stale copy plan", err)
	_, err = f.store.Put(f.ctx, f.ref(baseRef.Key), bytes.NewReader([]byte("changed externally")), int64(len("changed externally")), core.PutOptions{})
	requireNoError(t, "change source outside the service", err)
	_, err = svc.ExecutePlan(t.Context(), stalePlan.ID, true)
	requireErrorCode(t, err, "conflict")
	staleStored, err := svc.Plan(t.Context(), stalePlan.ID)
	requireNoError(t, "reload stale plan", err)
	if staleStored.Status != "pending" {
		t.Fatalf("Rejected plan changed its status to %q", staleStored.Status)
	}
	_, err = svc.Stat(t.Context(), core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/plans/stale.txt"})
	requireErrorCode(t, err, "not_found")

	uploadPath := writeFile(t, dir, "upload.txt", []byte("upload plan v1"))
	uploadRef := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/plans/upload.txt"}
	result, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: uploadRef, LocalPath: uploadPath})
	requireNoError(t, "seed upload plan object", err)
	waitTask(t, svc, result.Task.ID)
	writeFile(t, dir, "upload.txt", []byte("upload plan v2"))
	uploadPlan, err := svc.CreatePlan(t.Context(), core.PlanRequest{Operation: "upload", Target: uploadRef, LocalPath: uploadPath})
	requireNoError(t, "create upload plan", err)
	writeFile(t, dir, "upload.txt", []byte("upload plan v3 changed after planning"))
	_, err = svc.ExecutePlan(t.Context(), uploadPlan.ID, true)
	requireErrorCode(t, err, "conflict")
}

func testServiceReadOnly(t *testing.T, f *fixture, svc *service.Service, profile core.Profile) {
	readOnly := profile
	readOnly.ReadOnly = true
	saved, err := svc.SaveProfile(t.Context(), service.ProfileInput{Profile: readOnly})
	requireNoError(t, "save read-only profile", err)
	if !saved.ReadOnly {
		t.Fatal("Read-only flag was not saved")
	}
	path := writeFile(t, t.TempDir(), "readonly.txt", []byte("cannot upload"))
	ref := core.ObjectRef{ProfileID: profile.ID, Bucket: f.bucket, Key: "svc/readonly/blocked.txt"}
	_, err = svc.StartTransfer(t.Context(), core.TransferRequest{Direction: "upload", Object: ref, LocalPath: path})
	requireErrorCode(t, err, "read_only")
	_, err = svc.CreatePlan(t.Context(), core.PlanRequest{Operation: "delete", Sources: []core.ObjectRef{ref}})
	requireErrorCode(t, err, "read_only")
	_, err = svc.List(t.Context(), core.ListRequest{ProfileID: profile.ID, Bucket: f.bucket, Limit: 10})
	requireNoError(t, "list with read-only profile", err)

	profile.ReadOnly = false
	_, err = svc.SaveProfile(t.Context(), service.ProfileInput{Profile: profile})
	requireNoError(t, "restore writable profile", err)
}

func startService(t *testing.T, f *fixture) (*service.Service, core.Profile) {
	t.Helper()
	svc, err := service.New(t.Context(), service.Options{Dir: t.TempDir(), Vault: newTestVault(), Factory: f.factory})
	requireNoError(t, "create service", err)
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Error("Service cleanup failed; details suppressed")
		}
	})
	profile, err := svc.SaveProfile(t.Context(), service.ProfileInput{
		Profile: core.Profile{
			Name: "Integration " + f.profileID, ProviderType: "r2", AccountID: strings.Repeat("ab", 16),
			Bucket: f.bucket, CredentialSource: "keychain",
		},
		AccessKeyID: localAccessKey, SecretAccessKey: localSecretKey,
	})
	requireNoError(t, "save integration profile", err)
	if profile.ID == "" || profile.ProviderType != "r2" {
		t.Fatal("Saved profile is incomplete")
	}
	buckets, err := svc.TestConnection(t.Context(), profile.ID)
	requireNoError(t, "test service connection", err)
	if len(buckets) != 1 || buckets[0].Name != f.bucket {
		t.Fatalf("Connection test returned unexpected buckets")
	}
	return svc, profile
}

type testVault struct {
	mu     sync.Mutex
	values map[string]core.Credentials
}

func newTestVault() *testVault {
	return &testVault{values: map[string]core.Credentials{}}
}

func (v *testVault) Get(id string) (core.Credentials, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	creds, ok := v.values[id]
	if !ok {
		return core.Credentials{}, core.NewError("credentials_unavailable", "No saved integration credentials")
	}
	return creds, nil
}

func (v *testVault) Set(id string, creds core.Credentials) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.values[id] = creds
	return nil
}

func (v *testVault) Delete(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.values, id)
	return nil
}

func waitTask(t *testing.T, svc *service.Service, id string) core.TransferTask {
	t.Helper()
	if id == "" {
		t.Fatal("Transfer task ID is empty")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	for {
		tasks, err := svc.Tasks(ctx)
		requireNoError(t, "list transfer tasks", err)
		for _, task := range tasks {
			if task.ID != id {
				continue
			}
			switch task.Status {
			case "completed":
				return task
			case "failed", "cancelled":
				code := "unknown"
				if task.Error != nil {
					code = task.Error.Code
				}
				t.Fatalf("Transfer ended with status %s and code %s", task.Status, code)
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("Transfer did not finish before the test deadline")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal("Cannot write a test file")
	}
	return path
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	requireNoError(t, "read downloaded file", err)
	return data
}

func sha256File(t *testing.T, path string) string {
	t.Helper()
	return sha256Bytes(readFile(t, path))
}

func sha256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
