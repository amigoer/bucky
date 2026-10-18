//go:build integration

package integration_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/drivers/s3compat"
)

func TestBackendContract(t *testing.T) {
	for _, backend := range localBackends() {
		t.Run(backend.name, func(t *testing.T) {
			f := startFixture(t, backend)
			t.Run("list", func(t *testing.T) { testList(t, f) })
			t.Run("objects", func(t *testing.T) { testObjects(t, f) })
			t.Run("special keys", func(t *testing.T) { testSpecialKeys(t, f) })
			t.Run("copy", func(t *testing.T) { testCopy(t, f) })
			t.Run("multipart", func(t *testing.T) { testMultipart(t, f) })
			t.Run("presign", func(t *testing.T) { testPresign(t, f) })
			t.Run("delete", func(t *testing.T) { testDelete(t, f) })
			t.Run("auth errors", func(t *testing.T) { testAuthErrors(t, f) })
		})
	}
}

func testList(t *testing.T, f *fixture) {
	putBytes(t, f, "list/alpha.txt", []byte("alpha"), core.PutOptions{ContentType: "text/plain"})
	putBytes(t, f, "list/zero", nil, core.PutOptions{})
	putBytes(t, f, "list/dir one/beta.txt", []byte("beta"), core.PutOptions{})
	putBytes(t, f, "list/dir one/nested/文件 +%.txt", []byte("nested"), core.PutOptions{})
	putBytes(t, f, "list/dir two/gamma.txt", []byte("gamma"), core.PutOptions{})

	var objects []string
	prefixes := map[string]bool{}
	pages := 0
	cursor := ""
	for {
		page, err := f.store.List(f.ctx, core.ListRequest{ProfileID: f.profileID, Bucket: f.bucket, Prefix: "list/", Delimiter: "/", Cursor: cursor, Limit: 2})
		requireNoError(t, "list top-level page", err)
		pages++
		for _, object := range page.Objects {
			objects = append(objects, object.Key)
		}
		for _, prefix := range page.Prefixes {
			prefixes[prefix] = true
		}
		if page.NextCursor == "" {
			break
		}
		if page.NextCursor == cursor {
			t.Fatal("List repeated its cursor")
		}
		cursor = page.NextCursor
		if pages > 100 {
			t.Fatal("List did not terminate")
		}
	}
	if pages < 2 {
		t.Fatalf("Expected pagination, got %d page(s)", pages)
	}
	wantedObjects := map[string]bool{"list/alpha.txt": true, "list/zero": true}
	if len(objects) != len(wantedObjects) {
		t.Fatalf("Expected %d top-level objects, got %d", len(wantedObjects), len(objects))
	}
	for _, key := range objects {
		if !wantedObjects[key] {
			t.Fatalf("Unexpected top-level object %q", key)
		}
	}
	if !prefixes["list/dir one/"] || !prefixes["list/dir two/"] || len(prefixes) != 2 {
		t.Fatalf("Unexpected prefixes: %v", prefixes)
	}

	page, err := f.store.List(f.ctx, core.ListRequest{ProfileID: f.profileID, Bucket: f.bucket, Prefix: "list/dir one/", Delimiter: "/", Limit: 100})
	requireNoError(t, "list nested folder", err)
	if len(page.Objects) != 1 || page.Objects[0].Key != "list/dir one/beta.txt" {
		t.Fatalf("Unexpected nested objects: %v", page.Objects)
	}
	if len(page.Prefixes) != 1 || page.Prefixes[0] != "list/dir one/nested/" {
		t.Fatalf("Unexpected nested prefixes: %v", page.Prefixes)
	}

	page, err = f.store.List(f.ctx, core.ListRequest{ProfileID: f.profileID, Bucket: f.bucket, Prefix: "list/", Limit: 1000})
	requireNoError(t, "list recursive", err)
	if len(page.Objects) != 5 {
		t.Fatalf("Expected 5 recursive objects, got %d", len(page.Objects))
	}

	_, err = f.store.List(f.ctx, core.ListRequest{ProfileID: f.profileID, Bucket: f.bucket, Prefix: "list/", Limit: 1001})
	requireErrorCode(t, err, "invalid_argument")
}

func testObjects(t *testing.T, f *fixture) {
	content := []byte("hello bucky")
	info := putBytes(t, f, "objects/hello.txt", content, core.PutOptions{ContentType: "text/plain", Metadata: map[string]string{"owner": "bucky"}})
	if info.Size != int64(len(content)) || info.ETag == "" {
		t.Fatalf("Put returned incomplete metadata")
	}
	stat, err := f.store.Stat(f.ctx, f.ref("objects/hello.txt"))
	requireNoError(t, "stat object", err)
	if stat.Size != info.Size || stat.ETag != info.ETag || stat.ContentType != "text/plain" || stat.Metadata["owner"] != "bucky" {
		t.Fatalf("Stat metadata does not match Put")
	}

	reader, err := f.store.Read(f.ctx, f.ref("objects/hello.txt"), core.ReadOptions{})
	requireNoError(t, "read object", err)
	full := readBody(t, reader)
	if !bytes.Equal(full, content) {
		t.Fatal("Read returned different content")
	}

	reader, err = f.store.Read(f.ctx, f.ref("objects/hello.txt"), core.ReadOptions{Offset: 2, Length: 4, IfMatch: info.ETag})
	requireNoError(t, "read object range", err)
	partial := readBody(t, reader)
	if !bytes.Equal(partial, content[2:6]) {
		t.Fatalf("Unexpected range content %q", partial)
	}

	_, err = f.store.Read(f.ctx, f.ref("objects/hello.txt"), core.ReadOptions{IfMatch: "stale-etag"})
	requireErrorCode(t, err, "conflict")

	_, err = f.store.Read(f.ctx, f.ref("objects/missing.txt"), core.ReadOptions{})
	requireErrorCode(t, err, "not_found")

	created := putBytes(t, f, "objects/create-only.txt", []byte("first"), core.PutOptions{IfNoneMatch: true})
	_, err = f.store.Put(f.ctx, f.ref("objects/create-only.txt"), bytes.NewReader([]byte("second")), 6, core.PutOptions{IfNoneMatch: true})
	requireErrorCode(t, err, "conflict")

	putBytes(t, f, "objects/create-only.txt", []byte("updated"), core.PutOptions{IfMatch: created.ETag})
	current := statOf(t, f, "objects/create-only.txt")
	if current.Size != int64(len("updated")) {
		t.Fatalf("Conditional overwrite did not replace the object")
	}
	_, err = f.store.Put(f.ctx, f.ref("objects/create-only.txt"), bytes.NewReader([]byte("stale")), 5, core.PutOptions{IfMatch: created.ETag})
	requireErrorCode(t, err, "conflict")

	zero := putBytes(t, f, "objects/zero.bin", nil, core.PutOptions{ContentType: "application/octet-stream"})
	if zero.Size != 0 {
		t.Fatalf("Zero-byte object reported size %d", zero.Size)
	}
	reader, err = f.store.Read(f.ctx, f.ref("objects/zero.bin"), core.ReadOptions{})
	requireNoError(t, "read zero-byte object", err)
	if body := readBody(t, reader); len(body) != 0 {
		t.Fatalf("Zero-byte object returned %d bytes", len(body))
	}
}

func testSpecialKeys(t *testing.T, f *fixture) {
	key := "keys/a b+%/文件 #1.txt"
	copyKey := "keys/copied 文件 +%.txt"
	content := []byte("special key content")
	putBytes(t, f, key, content, core.PutOptions{ContentType: "text/plain; charset=utf-8"})

	stat, err := f.store.Stat(f.ctx, f.ref(key))
	requireNoError(t, "stat special key", err)
	reader, err := f.store.Read(f.ctx, f.ref(key), core.ReadOptions{IfMatch: stat.ETag})
	requireNoError(t, "read special key", err)
	if body := readBody(t, reader); !bytes.Equal(body, content) {
		t.Fatal("Special key content mismatch")
	}

	page, err := f.store.List(f.ctx, core.ListRequest{ProfileID: f.profileID, Bucket: f.bucket, Prefix: "keys/", Limit: 100})
	requireNoError(t, "list special keys", err)
	found := false
	for _, object := range page.Objects {
		if object.Key == key {
			found = true
		}
	}
	if !found {
		t.Fatalf("Special key was not listed")
	}

	copied, err := f.store.Copy(f.ctx, stat, f.ref(copyKey), core.PutOptions{IfNoneMatch: true})
	requireNoError(t, "copy special key", err)
	if copied.Size != stat.Size {
		t.Fatalf("Copied object has the wrong size")
	}
	reader, err = f.store.Read(f.ctx, f.ref(copyKey), core.ReadOptions{})
	requireNoError(t, "read copied special key", err)
	if body := readBody(t, reader); !bytes.Equal(body, content) {
		t.Fatal("Copied special key content mismatch")
	}
}

func testCopy(t *testing.T, f *fixture) {
	source := putBytes(t, f, "copy/source.txt", []byte("source"), core.PutOptions{ContentType: "text/plain", Metadata: map[string]string{"kind": "source"}})
	copied, err := f.store.Copy(f.ctx, source, f.ref("copy/target.txt"), core.PutOptions{IfNoneMatch: true})
	requireNoError(t, "copy object", err)
	if copied.Size != source.Size {
		t.Fatalf("Copied size mismatch")
	}
	target := statOf(t, f, "copy/target.txt")
	if target.ContentType != "text/plain" || target.Metadata["kind"] != "source" {
		t.Fatalf("Copy did not preserve content type and metadata")
	}

	_, err = f.store.Copy(f.ctx, source, f.ref("copy/target.txt"), core.PutOptions{IfNoneMatch: true})
	requireErrorCode(t, err, "conflict")

	overwritten, err := f.store.Copy(f.ctx, source, f.ref("copy/target.txt"), core.PutOptions{IfMatch: target.ETag})
	requireNoError(t, "conditional copy overwrite", err)
	if overwritten.ETag == "" {
		t.Fatal("Conditional copy returned no ETag")
	}

	stale := source
	stale.ETag = "stale-etag"
	_, err = f.store.Copy(f.ctx, stale, f.ref("copy/stale.txt"), core.PutOptions{IfNoneMatch: true})
	requireErrorCode(t, err, "conflict")
}

func testMultipart(t *testing.T, f *fixture) {
	data := makeMultipartData(12 << 20)
	partSize := int64(5 << 20)
	key := "multipart/object.bin"
	uploadID, err := f.store.CreateMultipart(f.ctx, f.ref(key), core.PutOptions{ContentType: "application/octet-stream", Metadata: map[string]string{"kind": "multipart"}})
	requireNoError(t, "create multipart upload", err)

	parts := []core.Part{}
	for number := int32(1); number <= 3; number++ {
		offset := int64(number-1) * partSize
		size := min(partSize, int64(len(data))-offset)
		part, err := f.store.UploadPart(f.ctx, f.ref(key), uploadID, number, bytes.NewReader(data[offset:offset+size]), size)
		requireNoError(t, "upload multipart part", err)
		if part.Number != number || part.Size != size || part.ETag == "" {
			t.Fatalf("UploadPart returned invalid metadata")
		}
		parts = append(parts, part)
	}
	remoteParts, err := f.store.ListParts(f.ctx, f.ref(key), uploadID)
	requireNoError(t, "list multipart parts", err)
	if len(remoteParts) != 3 {
		t.Fatalf("Expected 3 remote parts, got %d", len(remoteParts))
	}
	for i := range remoteParts {
		if remoteParts[i].Number != parts[i].Number || remoteParts[i].Size != parts[i].Size || remoteParts[i].ETag != parts[i].ETag {
			t.Fatalf("ListParts does not match uploaded parts")
		}
	}

	info, err := f.store.CompleteMultipart(f.ctx, f.ref(key), uploadID, parts, core.PutOptions{ContentType: "application/octet-stream", Metadata: map[string]string{"kind": "multipart"}})
	requireNoError(t, "complete multipart upload", err)
	if info.Size != int64(len(data)) || info.ETag == "" {
		t.Fatalf("Completed multipart metadata mismatch")
	}
	stat := statOf(t, f, key)
	if stat.Size != int64(len(data)) || stat.Metadata["kind"] != "multipart" || stat.ContentType != "application/octet-stream" {
		t.Fatalf("Completed multipart object metadata mismatch")
	}
	reader, err := f.store.Read(f.ctx, f.ref(key), core.ReadOptions{Offset: int64(len(data)) - 64, Length: 64, IfMatch: stat.ETag})
	requireNoError(t, "read completed multipart tail", err)
	if body := readBody(t, reader); !bytes.Equal(body, data[len(data)-64:]) {
		t.Fatal("Completed multipart content mismatch")
	}

	abortID, err := f.store.CreateMultipart(f.ctx, f.ref("multipart/aborted.bin"), core.PutOptions{})
	requireNoError(t, "create upload to abort", err)
	_, err = f.store.UploadPart(f.ctx, f.ref("multipart/aborted.bin"), abortID, 1, bytes.NewReader(data[:partSize]), partSize)
	requireNoError(t, "upload part to abort", err)
	requireNoError(t, "abort multipart upload", f.store.AbortMultipart(f.ctx, f.ref("multipart/aborted.bin"), abortID))
	_, err = f.store.ListParts(f.ctx, f.ref("multipart/aborted.bin"), abortID)
	requireErrorCode(t, err, "not_found")

	invalidID, err := f.store.CreateMultipart(f.ctx, f.ref("multipart/invalid.bin"), core.PutOptions{})
	requireNoError(t, "create invalid upload", err)
	_, err = f.store.CompleteMultipart(f.ctx, f.ref("multipart/invalid.bin"), invalidID, []core.Part{
		{Number: 1, ETag: `"one"`, Size: partSize},
		{Number: 2, ETag: `"two"`, Size: 1 << 20},
		{Number: 3, ETag: `"three"`, Size: 1 << 20},
	}, core.PutOptions{})
	requireErrorCode(t, err, "invalid_argument")
	requireNoError(t, "abort invalid upload", f.store.AbortMultipart(f.ctx, f.ref("multipart/invalid.bin"), invalidID))
}

func testPresign(t *testing.T, f *fixture) {
	content := []byte("presigned content")
	putBytes(t, f, "share/file.txt", content, core.PutOptions{ContentType: "text/plain"})
	link, err := f.store.Presign(f.ctx, f.ref("share/file.txt"), time.Hour)
	requireNoError(t, "presign object", err)
	if link.URL == "" || link.Public || link.ExpiresAt.Before(time.Now().Add(50*time.Minute)) || link.ExpiresAt.After(time.Now().Add(61*time.Minute)) {
		t.Fatalf("Presigned link metadata is invalid")
	}
	status, body := getPresigned(t, f.ctx, link, "", 1<<20)
	if status != 200 || !bytes.Equal(body, content) {
		t.Fatalf("Presigned GET returned status %d and %d bytes", status, len(body))
	}
	status, body = getPresigned(t, f.ctx, link, "bytes=0-3", 1<<20)
	if status != 206 || !bytes.Equal(body, content[:4]) {
		t.Fatalf("Presigned range GET returned status %d and %q", status, body)
	}
}

func testDelete(t *testing.T, f *fixture) {
	first := putBytes(t, f, "delete/first.txt", []byte("first"), core.PutOptions{})
	second := putBytes(t, f, "delete/second.txt", []byte("second"), core.PutOptions{})

	putBytes(t, f, "delete/first.txt", []byte("changed"), core.PutOptions{})
	results, err := f.store.Delete(f.ctx, []core.ObjectInfo{first})
	requireNoError(t, "delete stale object", err)
	if len(results) != 1 || results[0].Error == nil || results[0].Error.Code != "conflict" {
		t.Fatalf("Stale conditional delete did not fail with a conflict")
	}

	results, err = f.store.Delete(f.ctx, []core.ObjectInfo{second})
	requireNoError(t, "delete object", err)
	if len(results) != 1 || results[0].Error != nil {
		t.Fatalf("Delete reported an item error")
	}
	_, err = f.store.Stat(f.ctx, f.ref("delete/second.txt"))
	requireErrorCode(t, err, "not_found")

	results, err = f.store.Delete(f.ctx, []core.ObjectInfo{{ObjectRef: f.ref("delete/missing.txt"), ETag: `"missing"`}})
	requireNoError(t, "delete missing object", err)
	if len(results) != 1 || results[0].Error == nil || results[0].Error.Code != "not_found" {
		t.Fatalf("Deleting a missing object did not report not_found")
	}
}

func testAuthErrors(t *testing.T, f *fixture) {
	const profileID = "integration-bad-auth"
	store, err := s3compat.New(f.ctx, s3compat.Config{
		Endpoint:        f.endpoint,
		Region:          localRegion,
		PathStyle:       true,
		Credentials:     core.Credentials{AccessKeyID: "bad-access-key", SecretAccessKey: "bad-secret-key"},
		ProfileID:       profileID,
		ConditionPolicy: s3compat.R2Conditions,
	})
	requireNoError(t, "create store with bad credentials", err)
	_, err = store.List(f.ctx, core.ListRequest{ProfileID: profileID, Bucket: f.bucket, Limit: 1})
	if err == nil {
		t.Fatal("Expected an authentication error")
	}
	code := core.AsError(err).Code
	if code != "credentials_unavailable" && code != "permission_denied" {
		t.Fatalf("Unexpected authentication error code %q", code)
	}
}

func putBytes(t *testing.T, f *fixture, key string, data []byte, opts core.PutOptions) core.ObjectInfo {
	t.Helper()
	info, err := f.store.Put(f.ctx, f.ref(key), bytes.NewReader(data), int64(len(data)), opts)
	requireNoError(t, "put object", err)
	return info
}

func statOf(t *testing.T, f *fixture, key string) core.ObjectInfo {
	t.Helper()
	info, err := f.store.Stat(f.ctx, f.ref(key))
	requireNoError(t, "stat object", err)
	return info
}

func readBody(t *testing.T, reader *core.ObjectReader) []byte {
	t.Helper()
	if reader == nil || reader.Body == nil {
		t.Fatal("Read returned no body")
	}
	defer reader.Body.Close()
	body, err := io.ReadAll(io.LimitReader(reader.Body, 64<<20))
	requireNoError(t, "read object body", err)
	return body
}

func makeMultipartData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte((i*31 + 7) % 251)
	}
	return data
}
