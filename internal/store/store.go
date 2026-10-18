package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/amigoer/bucky/internal/core"
	_ "modernc.org/sqlite"
)

type Store struct {
	db  *sql.DB
	Dir string
}

func DataDir() (string, error) {
	if value := os.Getenv("BUCKY_DATA_DIR"); value != "" {
		return filepath.Abs(value)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Bucky"), nil
}

func Open(ctx context.Context, dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, core.NewError("storage", "Cannot create the application data directory")
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "bucky.db"))
	if err != nil {
		return nil, core.NewError("storage", "Cannot open the application database")
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, Dir: dir}
	_, err = db.ExecContext(ctx, `
		PRAGMA journal_mode=WAL;
		PRAGMA busy_timeout=5000;
		PRAGMA foreign_keys=ON;
		CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY);
		INSERT OR IGNORE INTO schema_version VALUES (1);
		CREATE TABLE IF NOT EXISTS profiles (id TEXT PRIMARY KEY, data TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS transfers (id TEXT PRIMARY KEY, status TEXT NOT NULL, data TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS transfer_parts (task_id TEXT NOT NULL REFERENCES transfers(id) ON DELETE CASCADE, number INTEGER NOT NULL, data TEXT NOT NULL, PRIMARY KEY(task_id, number));
		CREATE TABLE IF NOT EXISTS operation_plans (id TEXT PRIMARY KEY, status TEXT NOT NULL, data TEXT NOT NULL);
		CREATE TABLE IF NOT EXISTS usage_cache (id TEXT PRIMARY KEY, data TEXT NOT NULL);
	`)
	if err != nil {
		_ = db.Close()
		return nil, core.NewError("storage", "Cannot initialize the application database")
	}
	_ = os.Chmod(filepath.Join(dir, "bucky.db"), 0o600)
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) SaveProfile(ctx context.Context, p core.Profile) error {
	return s.put(ctx, "INSERT INTO profiles(id,data) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", p.ID, p)
}

func (s *Store) Profile(ctx context.Context, id string) (core.Profile, error) {
	var p core.Profile
	err := s.get(ctx, "SELECT data FROM profiles WHERE id=?", id, &p)
	return p, err
}

func (s *Store) Profiles(ctx context.Context) ([]core.Profile, error) {
	result := []core.Profile{}
	err := s.each(ctx, "SELECT data FROM profiles ORDER BY id", func(data []byte) error {
		var p core.Profile
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		result = append(result, p)
		return nil
	})
	return result, err
}

func (s *Store) DeleteProfile(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM profiles WHERE id=?", id)
	return storageError(err)
}

func (s *Store) SaveTask(ctx context.Context, t core.TransferTask) error {
	data, err := json.Marshal(t)
	if err != nil {
		return storageError(err)
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO transfers(id,status,data) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status,data=excluded.data", t.ID, t.Status, data)
	return storageError(err)
}

func (s *Store) Task(ctx context.Context, id string) (core.TransferTask, error) {
	var t core.TransferTask
	err := s.get(ctx, "SELECT data FROM transfers WHERE id=?", id, &t)
	return t, err
}

func (s *Store) Tasks(ctx context.Context) ([]core.TransferTask, error) {
	result := []core.TransferTask{}
	err := s.each(ctx, "SELECT data FROM transfers ORDER BY rowid DESC", func(data []byte) error {
		var t core.TransferTask
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
		result = append(result, t)
		return nil
	})
	return result, err
}

func (s *Store) SavePart(ctx context.Context, id string, p core.Part) error {
	data, err := json.Marshal(p)
	if err != nil {
		return storageError(err)
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO transfer_parts(task_id,number,data) VALUES(?,?,?) ON CONFLICT(task_id,number) DO UPDATE SET data=excluded.data", id, p.Number, data)
	return storageError(err)
}

func (s *Store) SaveTaskAndPart(ctx context.Context, t core.TransferTask, p core.Part) error {
	taskData, err := json.Marshal(t)
	if err != nil {
		return storageError(err)
	}
	partData, err := json.Marshal(p)
	if err != nil {
		return storageError(err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return storageError(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, "UPDATE transfers SET status=?,data=? WHERE id=?", t.Status, taskData, t.ID); err != nil {
		return storageError(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO transfer_parts(task_id,number,data) VALUES(?,?,?) ON CONFLICT(task_id,number) DO UPDATE SET data=excluded.data", t.ID, p.Number, partData); err != nil {
		return storageError(err)
	}
	return storageError(tx.Commit())
}

func (s *Store) Parts(ctx context.Context, id string) ([]core.Part, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT data FROM transfer_parts WHERE task_id=? ORDER BY number", id)
	if err != nil {
		return nil, storageError(err)
	}
	defer func() { _ = rows.Close() }()
	result := []core.Part{}
	for rows.Next() {
		var data []byte
		var p core.Part
		if err := rows.Scan(&data); err != nil {
			return nil, storageError(err)
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, storageError(err)
		}
		result = append(result, p)
	}
	return result, storageError(rows.Err())
}

func (s *Store) SavePlan(ctx context.Context, p core.OperationPlan) error {
	data, err := json.Marshal(p)
	if err != nil {
		return storageError(err)
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO operation_plans(id,status,data) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET status=excluded.status,data=excluded.data", p.ID, p.Status, data)
	return storageError(err)
}

func (s *Store) Plan(ctx context.Context, id string) (core.OperationPlan, error) {
	var p core.OperationPlan
	err := s.get(ctx, "SELECT data FROM operation_plans WHERE id=?", id, &p)
	return p, err
}

func (s *Store) ClaimPlan(ctx context.Context, p core.OperationPlan) error {
	p.Status = "running"
	data, err := json.Marshal(p)
	if err != nil {
		return storageError(err)
	}
	result, err := s.db.ExecContext(ctx, "UPDATE operation_plans SET status='running',data=? WHERE id=? AND status='pending'", data, p.ID)
	if err != nil {
		return storageError(err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return storageError(err)
	}
	if count != 1 {
		return core.NewError("conflict", "This plan has already been claimed")
	}
	return nil
}

func (s *Store) SaveUsage(ctx context.Context, u core.Usage) error {
	return s.put(ctx, "INSERT INTO usage_cache(id,data) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", u.ProfileID+"/"+u.Bucket+"/"+u.Prefix, u)
}

func (s *Store) Usage(ctx context.Context, id string) (core.Usage, error) {
	var u core.Usage
	err := s.get(ctx, "SELECT data FROM usage_cache WHERE id=?", id, &u)
	return u, err
}

// Recover is called only while the process owns the write lock.
func (s *Store) Recover(ctx context.Context) error {
	tasks, err := s.Tasks(ctx)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if task.Status == "running" || task.Status == "queued" || task.Status == "cancelling" {
			task.Status = "paused"
			task.BytesPerSecond = 0
			task.UpdatedAt = time.Now().UTC()
			if err := s.SaveTask(ctx, task); err != nil {
				return err
			}
		}
	}
	// An interrupted destructive operation is never replayed automatically.
	_, err = s.db.ExecContext(ctx, "UPDATE operation_plans SET status='interrupted', data=json_set(data,'$.status','interrupted') WHERE status='running'")
	return storageError(err)
}

func (s *Store) put(ctx context.Context, query, id string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return storageError(err)
	}
	_, err = s.db.ExecContext(ctx, query, id, data)
	return storageError(err)
}

func (s *Store) get(ctx context.Context, query, id string, value any) error {
	var data []byte
	if err := s.db.QueryRowContext(ctx, query, id).Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.NewError("not_found", "The requested local record does not exist")
		}
		return storageError(err)
	}
	return storageError(json.Unmarshal(data, value))
}

func (s *Store) each(ctx context.Context, query string, visit func([]byte) error) error {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return storageError(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return storageError(err)
		}
		if err := visit(data); err != nil {
			return storageError(err)
		}
	}
	return storageError(rows.Err())
}

func storageError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return core.AsError(err)
	}
	return core.NewError("storage", "The local database operation failed")
}
