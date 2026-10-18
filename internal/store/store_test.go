package store

import (
	"context"
	"testing"
	"time"

	"github.com/amigoer/bucky/internal/core"
)

func TestRecoveryRequiresExplicitInvocation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	task := core.TransferTask{ID: "interrupted", Status: "running", CreatedAt: time.Now()}
	if err := s.SaveTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePlan(ctx, core.OperationPlan{ID: "plan", Status: "running"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	before, err := s.Task(ctx, task.ID)
	if err != nil || before.Status != "running" {
		t.Fatalf("opening a database changed task state: %v %v", before.Status, err)
	}
	if err := s.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := s.Task(ctx, task.ID)
	if err != nil || after.Status != "paused" {
		t.Fatalf("recover task: %v %v", after.Status, err)
	}
	plan, err := s.Plan(ctx, "plan")
	if err != nil || plan.Status != "interrupted" {
		t.Fatalf("recover plan: %v %v", plan.Status, err)
	}
}

func TestPlanClaimOnlyOnce(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	p := core.OperationPlan{ID: "immutable-plan", Status: "pending"}
	if err := s.SavePlan(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimPlan(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := s.ClaimPlan(ctx, p); core.AsError(err).Code != "conflict" {
		t.Fatalf("second claim: %v", err)
	}
}

func TestPartAndProgressCommitTogether(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	task := core.TransferTask{ID: "upload", Status: "running", Completed: 0}
	if err := s.SaveTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	task.Completed = 8 * 1024 * 1024
	if err := s.SaveTaskAndPart(ctx, task, core.Part{Number: 1, ETag: "etag", Size: task.Completed}); err != nil {
		t.Fatal(err)
	}
	parts, err := s.Parts(ctx, task.ID)
	if err != nil || len(parts) != 1 {
		t.Fatalf("parts: %v %v", parts, err)
	}
	stored, err := s.Task(ctx, task.ID)
	if err != nil || stored.Completed != parts[0].Size {
		t.Fatalf("progress: %v %v", stored.Completed, err)
	}
}
