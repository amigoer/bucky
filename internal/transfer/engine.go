package transfer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"maps"
	"path/filepath"
	"sync"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/store"
	"github.com/gofrs/flock"
)

const (
	maxConcurrent            = 2
	multipartThreshold int64 = 16 << 20
	minPartSize        int64 = 8 << 20
	maxParts           int64 = 10000
)

type Engine struct {
	OnUpdate func(core.TransferTask)
	Writable func(context.Context, string) error

	db        *store.Store
	resolve   func(context.Context, string) (core.ObjectStore, error)
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	lock      *flock.Flock
	held      bool
	closed    bool
	fatal     error
	jobs      map[string]*job
	queue     []*job
	running   int
	wg        sync.WaitGroup
	events    chan core.TransferTask
	eventDone chan struct{}
	rate      limiter
}

type job struct {
	task    core.TransferTask
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	intent  string
	started bool
	err     error
}

func New(db *store.Store, resolve func(context.Context, string) (core.ObjectStore, error)) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		db: db, resolve: resolve, ctx: ctx, cancel: cancel,
		lock: flock.New(filepath.Join(db.Dir, "write.lock")),
		jobs: make(map[string]*job), events: make(chan core.TransferTask, 128),
		eventDone: make(chan struct{}),
	}
	go func() {
		defer close(e.eventDone)
		for task := range e.events {
			if e.OnUpdate != nil {
				e.OnUpdate(task)
			}
		}
	}()
	return e
}

// WithWriteLock callbacks must not call the engine's write methods.
func (e *Engine) WithWriteLock(ctx context.Context, fn func() error) error {
	if fn == nil {
		return core.Invalid("A write operation is required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.acquireLocked(ctx); err != nil {
		return err
	}
	err := fn()
	return errors.Join(err, e.releaseLocked())
}

func (e *Engine) Start(ctx context.Context, req core.TransferRequest, opts core.PutOptions) (core.TransferTask, error) {
	if req.Direction != "upload" && req.Direction != "download" {
		return core.TransferTask{}, core.Invalid("Direction must be upload or download")
	}
	if req.Object.ProfileID == "" || req.Object.Bucket == "" || req.Object.Key == "" || req.LocalPath == "" {
		return core.TransferTask{}, core.Invalid("A profile, bucket, key and local path are required")
	}
	if opts.IfMatch != "" && opts.IfNoneMatch {
		return core.TransferTask{}, core.Invalid("Upload conditions cannot be combined")
	}
	path, err := filepath.Abs(req.LocalPath)
	if err != nil {
		return core.TransferTask{}, core.Invalid("The local path is invalid")
	}
	task := core.TransferTask{
		ID: newID(), Direction: req.Direction, Object: req.Object,
		LocalPath: filepath.Clean(path), Status: "queued",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if req.Direction == "upload" {
		if err := e.checkWritable(ctx, req.Object.ProfileID); err != nil {
			return core.TransferTask{}, err
		}
		file, err := openSource(task.LocalPath)
		if err != nil {
			return core.TransferTask{}, err
		}
		info, digest, err := fingerprint(ctx, file)
		closeErr := file.Close()
		if err != nil {
			return core.TransferTask{}, err
		}
		if closeErr != nil {
			return core.TransferTask{}, localError("Cannot close the upload source")
		}
		if req.ExpectedSHA256 != "" && req.ExpectedSHA256 != digest {
			return core.TransferTask{}, core.NewError("conflict", "The upload source changed after the plan was created")
		}
		task.Size, task.SourceModTime, task.SourceSHA256 = info.Size(), info.ModTime().UnixNano(), digest
		if task.Size > multipartThreshold {
			task.PartSize = max(minPartSize, (task.Size+maxParts-1)/maxParts)
		}
		if opts.IfMatch == "" {
			opts.IfNoneMatch = true
		}
		opts.Metadata = maps.Clone(opts.Metadata)
		if opts.Metadata == nil {
			opts.Metadata = make(map[string]string)
		}
		opts.Metadata["bucky-transfer-id"] = task.ID
		opts.Metadata["bucky-sha256"] = task.SourceSHA256
		task.PutOptions, task.IfMatch = opts, opts.IfMatch
		task.Overwrite = !opts.IfNoneMatch
	} else if err := checkDestination(task.LocalPath); err != nil {
		return core.TransferTask{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.acquireLocked(ctx); err != nil {
		return core.TransferTask{}, err
	}
	if err := e.saveLocked(task); err != nil {
		return core.TransferTask{}, errors.Join(err, e.releaseLocked())
	}
	e.enqueueLocked(task, "")
	return cloneTask(task), nil
}

func (e *Engine) Resume(ctx context.Context, id string) (core.TransferTask, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.acquireLocked(ctx); err != nil {
		return core.TransferTask{}, err
	}
	if _, ok := e.jobs[id]; ok {
		return core.TransferTask{}, core.NewError("task_busy", "The transfer is already active")
	}
	task, err := e.db.Task(ctx, id)
	if err == nil && task.Status != "paused" && task.Status != "failed" {
		err = core.NewError("conflict", "Only a paused or failed transfer can be resumed")
	}
	if err == nil && task.Direction == "upload" {
		err = e.checkWritable(ctx, task.Object.ProfileID)
	}
	if err == nil && task.Direction == "upload" {
		file, openErr := openSource(task.LocalPath)
		if openErr != nil {
			err = openErr
		} else {
			_, err = verifySource(ctx, file, task)
			err = errors.Join(err, file.Close())
		}
	}
	if err == nil && task.Direction == "download" {
		err = checkDestination(task.LocalPath)
	}
	if err != nil {
		return core.TransferTask{}, errors.Join(err, e.releaseLocked())
	}
	task.Status, task.Error, task.BytesPerSecond = "queued", nil, 0
	task.UpdatedAt = time.Now().UTC()
	if err := e.saveLocked(task); err != nil {
		return core.TransferTask{}, errors.Join(err, e.releaseLocked())
	}
	e.enqueueLocked(task, "")
	return cloneTask(task), nil
}

func (e *Engine) Pause(ctx context.Context, id string) error {
	return e.stop(ctx, id, "pause")
}

func (e *Engine) Cancel(ctx context.Context, id string) error {
	return e.stop(ctx, id, "cancel")
}

func (e *Engine) stop(ctx context.Context, id, intent string) error {
	e.mu.Lock()
	if err := e.acquireLocked(ctx); err != nil {
		e.mu.Unlock()
		return err
	}
	j, active := e.jobs[id]
	if !active {
		task, err := e.db.Task(ctx, id)
		if err != nil {
			err = errors.Join(err, e.releaseLocked())
			e.mu.Unlock()
			return err
		}
		if intent == "pause" && task.Status == "paused" || intent == "cancel" && task.Status == "cancelled" {
			err = e.releaseLocked()
			e.mu.Unlock()
			return err
		}
		if intent == "pause" || task.Status == "completed" {
			err = errors.Join(core.NewError("conflict", "The transfer is not active"), e.releaseLocked())
			e.mu.Unlock()
			return err
		}
		if task.Direction == "upload" {
			if err := e.checkWritable(ctx, task.Object.ProfileID); err != nil {
				err = errors.Join(err, e.releaseLocked())
				e.mu.Unlock()
				return err
			}
		}
		task.Status, task.UpdatedAt = "cancelling", time.Now().UTC()
		if err := e.saveLocked(task); err != nil {
			err = errors.Join(err, e.releaseLocked())
			e.mu.Unlock()
			return err
		}
		j = e.enqueueLocked(task, "cancel")
	} else {
		if j.intent == "cancel" && intent == "pause" {
			e.mu.Unlock()
			return core.NewError("conflict", "The transfer is being cancelled")
		}
		j.intent = intent
		if intent == "cancel" {
			task := j.task
			task.Status, task.UpdatedAt = "cancelling", time.Now().UTC()
			if err := e.saveLocked(task); err != nil {
				e.mu.Unlock()
				return err
			}
			j.task = task
		}
		if j.started {
			j.cancel()
		} else if intent == "pause" {
			task := j.task
			task.Status, task.BytesPerSecond, task.UpdatedAt = "paused", 0, time.Now().UTC()
			if err := e.saveLocked(task); err != nil {
				e.mu.Unlock()
				return err
			}
			j.task = task
			delete(e.jobs, id)
			j.cancel()
			close(j.done)
			j.err = e.releaseLocked()
		}
	}
	e.mu.Unlock()
	select {
	case <-ctx.Done():
		return core.AsError(ctx.Err())
	case <-j.done:
		return j.err
	}
}

func (e *Engine) Tasks(ctx context.Context) ([]core.TransferTask, error) {
	return e.db.Tasks(ctx)
}

func (e *Engine) SetRate(bytesPerSecond int64) {
	e.rate.set(bytesPerSecond)
}

func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		err := e.fatal
		e.mu.Unlock()
		return err
	}
	e.closed = true
	for id, j := range e.jobs {
		if j.started {
			if j.intent != "cancel" {
				j.intent = "pause"
			}
			j.cancel()
			continue
		}
		j.task.Status, j.task.BytesPerSecond, j.task.UpdatedAt = "paused", 0, time.Now().UTC()
		j.err = e.saveLocked(j.task)
		delete(e.jobs, id)
		j.cancel()
		close(j.done)
	}
	e.queue = nil
	e.cancel()
	e.mu.Unlock()
	e.wg.Wait()
	e.mu.Lock()
	err := errors.Join(e.fatal, e.releaseLocked())
	e.mu.Unlock()
	close(e.events)
	<-e.eventDone
	return err
}

func (e *Engine) checkWritable(ctx context.Context, id string) error {
	if e.Writable != nil {
		return e.Writable(ctx, id)
	}
	return nil
}

func (e *Engine) acquireLocked(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return core.AsError(err)
	}
	if e.closed {
		return core.NewError("closed", "The transfer engine is closed")
	}
	if e.fatal != nil {
		return e.fatal
	}
	if e.held {
		return nil
	}
	locked, err := e.lock.TryLock()
	if err != nil {
		return core.NewError("storage", "Cannot acquire the application write lock")
	}
	if !locked {
		return core.NewError("task_busy", "Another process is performing a write operation")
	}
	e.held = true
	if len(e.jobs) == 0 {
		if err := e.db.Recover(ctx); err != nil {
			return errors.Join(err, e.releaseLocked())
		}
	}
	return nil
}

func (e *Engine) releaseLocked() error {
	if !e.held || len(e.jobs) != 0 {
		return nil
	}
	if err := e.lock.Unlock(); err != nil {
		err = core.NewError("storage", "Cannot release the application write lock")
		e.fatal = errors.Join(e.fatal, err)
		return err
	}
	e.held = false
	return nil
}

func (e *Engine) saveLocked(task core.TransferTask) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.db.SaveTask(ctx, task); err != nil {
		e.fatal = errors.Join(e.fatal, err)
		return err
	}
	e.emit(task)
	return nil
}

func (e *Engine) emit(task core.TransferTask) {
	select {
	case e.events <- cloneTask(task):
	default:
	}
}

func (e *Engine) enqueueLocked(task core.TransferTask, intent string) *job {
	ctx, cancel := context.WithCancel(e.ctx)
	j := &job{task: task, ctx: ctx, cancel: cancel, done: make(chan struct{}), intent: intent}
	e.jobs[task.ID] = j
	e.queue = append(e.queue, j)
	e.scheduleLocked()
	return j
}

func (e *Engine) scheduleLocked() {
	for !e.closed && e.running < maxConcurrent && len(e.queue) != 0 {
		j := e.queue[0]
		e.queue = e.queue[1:]
		if e.jobs[j.task.ID] != j {
			continue
		}
		j.started = true
		e.running++
		e.wg.Go(func() { e.run(j) })
	}
}

func (e *Engine) run(j *job) {
	e.mu.Lock()
	t := &execution{engine: e, job: j, task: j.task, started: time.Now(), base: j.task.Completed}
	intent := j.intent
	e.mu.Unlock()
	var err error
	if intent != "cancel" {
		t.task.Status = "running"
		err = t.save()
		if err == nil && t.task.Direction == "upload" {
			err = e.checkWritable(j.ctx, t.task.Object.ProfileID)
		}
		if err == nil {
			var remote core.ObjectStore
			remote, err = e.resolve(j.ctx, t.task.Object.ProfileID)
			if err == nil && t.task.Direction == "upload" {
				err = t.upload(remote)
			} else if err == nil {
				err = t.download(remote)
			}
		}
	}
	e.mu.Lock()
	intent = j.intent
	e.mu.Unlock()
	if intent == "cancel" {
		cleanupErr := t.cleanup()
		if cleanupErr != nil {
			t.task.Status, t.task.Error = "failed", core.NewError("cleanup_failed", "Transfer cancellation could not clean up all temporary data")
			t.task.Error.Retryable = true
			err = errors.Join(err, cleanupErr, t.task.Error)
		} else {
			t.task.Status, t.task.Error, err = "cancelled", nil, nil
		}
	} else if err != nil {
		if intent == "pause" || errors.Is(err, context.Canceled) {
			t.task.Status, t.task.Error = "paused", nil
		} else {
			t.task.Status, t.task.Error = "failed", core.AsError(err)
		}
	} else {
		t.task.Status, t.task.Error, t.task.Completed = "completed", nil, t.task.Size
	}
	t.task.BytesPerSecond, t.task.UpdatedAt = 0, time.Now().UTC()
	e.mu.Lock()
	persistErr := e.saveLocked(t.task)
	j.task = t.task
	j.err = persistErr
	if intent == "cancel" && t.task.Status == "failed" {
		j.err = errors.Join(j.err, err)
	}
	delete(e.jobs, t.task.ID)
	e.running--
	j.cancel()
	e.scheduleLocked()
	j.err = errors.Join(j.err, e.releaseLocked())
	close(j.done)
	e.mu.Unlock()
}

func newID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic("secure random source unavailable")
	}
	return hex.EncodeToString(raw[:])
}

func cloneTask(task core.TransferTask) core.TransferTask {
	task.PutOptions.Metadata = maps.Clone(task.PutOptions.Metadata)
	if task.Error != nil {
		task.Error = new(*task.Error)
	}
	return task
}

type execution struct {
	engine   *Engine
	job      *job
	task     core.TransferTask
	started  time.Time
	base     int64
	lastSave time.Time
}

func (t *execution) save() error {
	t.task.UpdatedAt = time.Now().UTC()
	t.engine.mu.Lock()
	defer t.engine.mu.Unlock()
	if t.job.intent == "cancel" {
		t.task.Status = "cancelling"
	}
	if err := t.engine.saveLocked(t.task); err != nil {
		return err
	}
	t.job.task = t.task
	t.lastSave = time.Now()
	return nil
}

func (t *execution) savePart(part core.Part) error {
	t.task.UpdatedAt = time.Now().UTC()
	t.engine.mu.Lock()
	defer t.engine.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if t.job.intent == "cancel" {
		t.task.Status = "cancelling"
	}
	if err := t.engine.db.SaveTaskAndPart(ctx, t.task, part); err != nil {
		t.engine.fatal = errors.Join(t.engine.fatal, err)
		return err
	}
	t.job.task = t.task
	t.lastSave = time.Now()
	t.engine.emit(t.task)
	return nil
}

func (t *execution) progress(completed int64, force bool) error {
	t.task.Completed = completed
	if elapsed := time.Since(t.started).Seconds(); elapsed > 0 {
		t.task.BytesPerSecond = int64(float64(max(0, completed-t.base)) / elapsed)
	}
	if force || time.Since(t.lastSave) >= 500*time.Millisecond {
		return t.save()
	}
	return nil
}
