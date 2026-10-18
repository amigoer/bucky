package transfer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amigoer/bucky/internal/core"
)

func TestOpenSourceAllowsSymlinkedParent(t *testing.T) {
	dir := t.TempDir()
	real, err := os.MkdirTemp(dir, "real")
	if err != nil {
		t.Fatal("Cannot create the test directory")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("Symbolic links are unavailable")
	}
	target := filepath.Join(real, "file.txt")
	if err := os.WriteFile(target, []byte("content"), 0o600); err != nil {
		t.Fatal("Cannot write the test file")
	}
	file, err := openSource(filepath.Join(link, "file.txt"))
	if err != nil {
		t.Fatalf("A regular file below a symlinked directory was rejected: %s", core.AsError(err).Code)
	}
	if err := file.Close(); err != nil {
		t.Fatal("Cannot close the opened source")
	}
}

func TestOpenSourceRejectsFileSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("content"), 0o600); err != nil {
		t.Fatal("Cannot write the test file")
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("Symbolic links are unavailable")
	}
	_, err := openSource(link)
	if err == nil || core.AsError(err).Code != "local_file" {
		t.Fatal("A symlinked upload source was not rejected")
	}
}

func TestOpenSourceRejectsRelativePath(t *testing.T) {
	_, err := openSource("relative.txt")
	if err == nil || core.AsError(err).Code != "local_file" {
		t.Fatal("A relative upload source was not rejected")
	}
}

func TestCheckDestination(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.bin")
	if err := checkDestination(missing); err != nil {
		t.Fatalf("A missing destination was rejected: %s", core.AsError(err).Code)
	}
	existing := filepath.Join(dir, "existing.bin")
	if err := os.WriteFile(existing, []byte("existing"), 0o600); err != nil {
		t.Fatal("Cannot write the test file")
	}
	err := checkDestination(existing)
	if err == nil || core.AsError(err).Code != "conflict" {
		t.Fatal("An existing destination was not rejected")
	}
}
