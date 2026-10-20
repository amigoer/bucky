package desktop

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSidebarSlides(t *testing.T) {
	const heading = "A little space for everything"
	app := newTestApp(t)
	tt := ui.NewTester(app.view, 1280, 840)
	tt.Frame()
	open, ok := tt.Find(heading)
	if !ok {
		t.Fatal("the welcome page shows no heading")
	}

	app.sidebarOpen = false
	tt.Frame()
	tt.Advance(sidebarSlide / 2)
	half, _ := tt.Find(heading)
	tt.Advance(sidebarSlide)
	closed, _ := tt.Find(heading)
	if !(closed.X < half.X && half.X < open.X) {
		t.Errorf("heading at x %v, %v, %v while the sidebar closes; want it to move left in steps",
			open.X, half.X, closed.X)
	}
	if tt.HasText("Quick upload") {
		t.Error("the sidebar is still built after it slid away")
	}

	app.sidebarOpen = true
	tt.Frame()
	tt.Advance(sidebarSlide / 2)
	half, _ = tt.Find(heading)
	tt.Advance(sidebarSlide)
	reopened, _ := tt.Find(heading)
	if !(closed.X < half.X && half.X < reopened.X) || reopened != open {
		t.Errorf("heading at x %v, %v, %v while the sidebar opens; want it back at %v in steps",
			closed.X, half.X, reopened.X, open.X)
	}
}

func TestSegmentThumbSlides(t *testing.T) {
	app := newTestApp(t)
	app.page = pageTransfers
	app.transfers.loaded = true
	tt := ui.NewTester(app.view, 1280, 840)
	tt.Frame()
	first, ok := tt.Find("All")
	last, ok2 := tt.Find("Needs attention")
	if !ok || !ok2 {
		t.Fatal("the transfer filters are missing")
	}
	// The thumb is the only white in the track, along a line above the
	// labels' glyphs.
	thumbCenter := func() float32 {
		img := tt.Image()
		y := int(first.Y) - 2
		lo, hi := -1, -1
		for x := int(first.X) - 10; x < int(last.X+last.W)+10; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r>>8 >= 254 && g>>8 >= 254 && b>>8 >= 254 {
				if lo < 0 {
					lo = x
				}
				hi = x
			}
		}
		if lo < 0 {
			t.Fatal("no thumb in the filter track")
		}
		return float32(lo+hi) / 2
	}
	from, to := first.X+first.W/2, last.X+last.W/2
	if got := thumbCenter(); got < from-3 || got > from+3 {
		t.Fatalf("thumb centered at %v, want it on All at %v", got, from)
	}

	if err := tt.Click("Needs attention"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(segmentSlide / 2)
	if got := thumbCenter(); got < from+10 || got > to-10 {
		t.Errorf("thumb centered at %v halfway, want it between %v and %v", got, from, to)
	}
	tt.Advance(segmentSlide)
	if got := thumbCenter(); got < to-3 || got > to+3 {
		t.Errorf("thumb centered at %v after the slide, want it on Needs attention at %v", got, to)
	}
}
