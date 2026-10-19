package transfer

import (
	"context"
	"testing"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/store"
)

func TestSetConcurrencyClamps(t *testing.T) {
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	e := New(db, func(context.Context, string) (core.ObjectStore, error) {
		return nil, core.NewError("unsupported", "none")
	})
	defer func() { _ = e.Close() }()
	for _, c := range []struct{ in, want int }{{0, 1}, {-3, 1}, {4, 4}, {8, 8}, {64, 8}} {
		if got := e.SetConcurrency(c.in); got != c.want {
			t.Fatalf("SetConcurrency(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
