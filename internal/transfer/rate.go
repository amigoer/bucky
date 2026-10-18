package transfer

import (
	"context"
	"io"
	"sync"
	"time"
)

type limiter struct {
	mu   sync.Mutex
	rate int64
	next time.Time
}

func (l *limiter) set(rate int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rate, l.next = max(0, rate), time.Time{}
}

func (l *limiter) wait(ctx context.Context, bytes int) error {
	l.mu.Lock()
	if l.rate == 0 || bytes == 0 {
		l.mu.Unlock()
		return ctx.Err()
	}
	now := time.Now()
	start := l.next
	if start.Before(now) {
		start = now
	}
	l.next = start.Add(time.Duration(float64(bytes) / float64(l.rate) * float64(time.Second)))
	l.mu.Unlock()
	delay := time.Until(l.next)
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type uploadReader struct {
	ctx      context.Context
	reader   io.Reader
	limiter  *limiter
	count    int64
	progress func(int64) error
}

func (r *uploadReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	p = p[:min(len(p), 64<<10)]
	n, err := r.reader.Read(p)
	if n != 0 {
		if waitErr := r.limiter.wait(r.ctx, n); waitErr != nil {
			return n, waitErr
		}
		r.count += int64(n)
		if progressErr := r.progress(r.count); progressErr != nil {
			return n, progressErr
		}
	}
	return n, err
}
