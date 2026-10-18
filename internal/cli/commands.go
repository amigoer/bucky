package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amigoer/bucky/internal/agent"
	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/spf13/cobra"
)

func (a *app) profilesCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "profiles", Short: "Manage saved R2 connections"}
	cmd.AddCommand(
		&cobra.Command{Use: "list", Short: "List saved connections", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			return a.execute(cmd, func(ctx context.Context) (any, error) { return a.svc.Profiles(ctx) })
		}},
		a.profileAddCommand(),
		&cobra.Command{Use: "test ID", Short: "Test a connection and list visible buckets", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			return a.execute(cmd, func(ctx context.Context) (any, error) {
				id, err := a.profile(ctx, args[0])
				if err != nil {
					return nil, err
				}
				return a.svc.TestConnection(ctx, id)
			})
		}},
		a.profileRemoveCommand(),
		&cobra.Command{Use: "show ID", Short: "Show connection metadata", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			return a.execute(cmd, func(ctx context.Context) (any, error) {
				profiles, err := a.svc.Profiles(ctx)
				if err != nil {
					return nil, err
				}
				id, err := a.profile(ctx, args[0])
				if err != nil {
					return nil, err
				}
				for _, p := range profiles {
					if p.ID == id {
						return p, nil
					}
				}
				return nil, core.NewError("not_found", "The connection was not found")
			})
		}},
	)
	return cmd
}

func (a *app) profileAddCommand() *cobra.Command {
	var input service.ProfileInput
	var environment bool
	cmd := &cobra.Command{
		Use: "add", Short: "Add an R2 connection", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateFlags(cmd); err != nil {
				return err
			}
			if environment {
				input.Profile.CredentialSource = "environment"
			} else {
				input.Profile.CredentialSource = "keychain"
				var err error
				input.AccessKeyID, err = readSecret("R2 S3 Access Key ID: ")
				if err != nil {
					return core.Invalid("Credentials must be entered interactively or supplied through the documented process environment")
				}
				input.SecretAccessKey, err = readSecret("R2 S3 Secret Access Key: ")
				if err != nil {
					return core.Invalid("Credentials must be entered interactively or supplied through the documented process environment")
				}
			}
			return a.execute(cmd, func(ctx context.Context) (any, error) { return a.svc.SaveProfile(ctx, input) })
		},
	}
	f := cmd.Flags()
	f.StringVar(&input.Profile.Name, "name", "", "Connection name")
	f.StringVar(&input.Profile.AccountID, "account", "", "Cloudflare account ID")
	f.StringVar(&input.Profile.Bucket, "bucket", "", "Restrict the connection to one bucket")
	f.StringVar(&input.Profile.Jurisdiction, "jurisdiction", "default", "R2 jurisdiction: default, eu or fedramp")
	f.StringVar(&input.Profile.PublicURL, "public-url", "", "Existing public bucket URL")
	f.BoolVar(&input.Profile.ReadOnly, "readonly", false, "Block all writes through this connection")
	f.BoolVar(&environment, "environment", false, "Read BUCKY_R2_* credentials from this process instead of the keychain")
	return cmd
}

func (a *app) profileRemoveCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "remove ID", Short: "Remove a local connection and its keychain entry", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !yes {
			return core.NewError("confirmation_required", "Review the connection and rerun with --yes to remove its local configuration")
		}
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			id, err := a.profile(ctx, args[0])
			if err != nil {
				return nil, err
			}
			if err := a.svc.RemoveProfile(ctx, id, true); err != nil {
				return nil, err
			}
			return "Connection removed.", nil
		})
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm local connection removal")
	return cmd
}

func (a *app) lsCommand() *cobra.Command {
	var cursor string
	var limit int32
	var recursive bool
	cmd := &cobra.Command{Use: "ls r2://profile/bucket/prefix", Short: "List objects and folders", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			ref, err := a.resolveRemote(ctx, args[0], false)
			if err != nil {
				return nil, err
			}
			delimiter := "/"
			if recursive {
				delimiter = ""
			}
			return a.svc.List(ctx, core.ListRequest{ProfileID: ref.ProfileID, Bucket: ref.Bucket, Prefix: ref.Key, Delimiter: delimiter, Cursor: cursor, Limit: limit})
		})
	}}
	cmd.Flags().StringVar(&cursor, "cursor", "", "Continuation cursor")
	cmd.Flags().Int32Var(&limit, "limit", 200, "Page size from 1 to 1000")
	cmd.Flags().BoolVar(&recursive, "recursive", false, "List keys without grouping folders")
	return cmd
}

func (a *app) statCommand() *cobra.Command {
	return &cobra.Command{Use: "stat r2://profile/bucket/key", Short: "Read object metadata", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			ref, err := a.resolveRemote(ctx, args[0], true)
			if err != nil {
				return nil, err
			}
			return a.svc.Stat(ctx, ref)
		})
	}}
}

func (a *app) findCommand() *cobra.Command {
	var req core.FindRequest
	cmd := &cobra.Command{Use: "find r2://profile/bucket/prefix", Short: "Search a prefix; reports scan completeness", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := agent.ValidateFind(req.Before, req.MinSize, req.MaxSize, req.Limit); err != nil {
			return err
		}
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			ref, err := a.resolveRemote(ctx, args[0], false)
			if err != nil {
				return nil, err
			}
			req.ProfileID, req.Bucket, req.Prefix = ref.ProfileID, ref.Bucket, ref.Key
			return a.svc.Find(ctx, req)
		})
	}}
	f := cmd.Flags()
	f.StringVar(&req.Contains, "contains", "", "Case-insensitive key substring")
	f.StringVar(&req.Before, "before", "", "Modified before this RFC3339 timestamp")
	f.Int64Var(&req.MinSize, "min-size", 0, "Minimum size in bytes")
	f.Int64Var(&req.MaxSize, "max-size", 0, "Maximum size in bytes (0 means no maximum)")
	f.IntVar(&req.Limit, "limit", 1000, "Result threshold")
	f.StringVar(&req.Cursor, "cursor", "", "Continuation cursor")
	return cmd
}

func (a *app) presignCommand() *cobra.Command {
	var seconds int64
	var public bool
	cmd := &cobra.Command{Use: "presign r2://profile/bucket/key", Short: "Create a temporary read link", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			ref, err := a.resolveRemote(ctx, args[0], true)
			if err != nil {
				return nil, err
			}
			return a.svc.Share(ctx, ref, seconds, public)
		})
	}}
	cmd.Flags().Int64Var(&seconds, "seconds", 3600, "Link lifetime; R2 allows up to 604800 seconds")
	cmd.Flags().BoolVar(&public, "public", false, "Use the configured public bucket domain")
	return cmd
}

func (a *app) duCommand() *cobra.Command {
	return &cobra.Command{Use: "du r2://profile/bucket/prefix", Short: "Count objects and bytes under a prefix", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			ref, err := a.resolveRemote(ctx, args[0], false)
			if err != nil {
				return nil, err
			}
			return a.svc.Usage(ctx, ref.ProfileID, ref.Bucket, ref.Key)
		})
	}}
}

func (a *app) cpCommand() *cobra.Command {
	var yes, recursive bool
	cmd := &cobra.Command{Use: "cp SOURCE DEST", Short: "Copy between local storage and R2", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			return a.copy(ctx, args[0], args[1], yes, recursive)
		})
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "Authorize execution of any required overwrite plan")
	cmd.Flags().BoolVar(&recursive, "recursive", false, "Upload a directory tree")
	return cmd
}

func (a *app) copy(ctx context.Context, source, destination string, yes, recursive bool) (any, error) {
	srcRemote := strings.HasPrefix(source, "r2://")
	dstRemote := strings.HasPrefix(destination, "r2://")
	switch {
	case !srcRemote && !dstRemote:
		return nil, core.Invalid("At least one side must be an r2:// remote")
	case srcRemote && dstRemote:
		src, err := a.resolveRemote(ctx, source, true)
		if err != nil {
			return nil, err
		}
		dst, err := a.resolveRemote(ctx, destination, true)
		if err != nil {
			return nil, err
		}
		plan, err := a.svc.CreatePlan(ctx, core.PlanRequest{Operation: "copy", Sources: []core.ObjectRef{src}, Target: dst})
		if err != nil {
			return nil, err
		}
		if !yes {
			if err := a.print(plan); err != nil {
				return nil, err
			}
			return nil, core.NewError("confirmation_required", "Review the plan and rerun with --yes to execute it")
		}
		return a.svc.ExecutePlan(ctx, plan.ID, true)
	case dstRemote:
		dst, err := a.resolveRemote(ctx, destination, true)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(source)
		if err != nil {
			return nil, core.Invalid("The local source does not exist")
		}
		if info.IsDir() {
			if !recursive {
				return nil, core.Invalid("Use --recursive to upload a directory")
			}
			return a.uploadDirectory(ctx, source, dst, yes)
		}
		task, err := a.uploadFile(ctx, source, dst, yes)
		if err != nil {
			return nil, err
		}
		return a.waitTask(ctx, task.ID)
	default:
		src, err := a.resolveRemote(ctx, source, true)
		if err != nil {
			return nil, err
		}
		filename := destination
		if info, err := os.Stat(destination); err == nil && info.IsDir() {
			filename = filepath.Join(destination, filepath.Base(src.Key))
		}
		task, err := a.svc.StartTransfer(ctx, core.TransferRequest{Direction: "download", Object: src, LocalPath: filename})
		if err != nil {
			return nil, err
		}
		if task.Task == nil {
			return nil, core.NewError("internal", "The download did not create a transfer task")
		}
		return a.waitTask(ctx, task.Task.ID)
	}
}

func (a *app) uploadFile(ctx context.Context, filename string, target core.ObjectRef, yes bool) (core.TransferTask, error) {
	result, err := a.svc.StartTransfer(ctx, core.TransferRequest{Direction: "upload", Object: target, LocalPath: filename})
	if err != nil {
		return core.TransferTask{}, err
	}
	if result.Plan != nil {
		if !yes {
			if err := a.print(*result.Plan); err != nil {
				return core.TransferTask{}, err
			}
			return core.TransferTask{}, core.NewError("confirmation_required", "Review the overwrite plan and rerun with --yes to execute it")
		}
		plan, err := a.svc.ExecutePlan(ctx, result.Plan.ID, true)
		if err != nil {
			return core.TransferTask{}, err
		}
		if len(plan.Results) != 1 || !strings.HasPrefix(plan.Results[0].Status, "task:") {
			return core.TransferTask{}, core.NewError("internal", "The upload plan did not create a transfer")
		}
		return a.svc.DB.Task(ctx, strings.TrimPrefix(plan.Results[0].Status, "task:"))
	}
	if result.Task == nil {
		return core.TransferTask{}, core.NewError("internal", "The upload did not create a transfer task")
	}
	return *result.Task, nil
}

func (a *app) uploadDirectory(ctx context.Context, root string, target core.ObjectRef, yes bool) (any, error) {
	base := filepath.Clean(root)
	prefix := target.Key
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	results := []core.TransferTask{}
	plans := []core.OperationPlan{}
	err := filepath.WalkDir(base, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return core.NewError("local_io", "Cannot read the local directory")
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return core.Invalid("Symbolic links are not uploaded")
		}
		rel, err := filepath.Rel(base, filename)
		if err != nil {
			return core.Invalid("Cannot determine the upload key")
		}
		key := prefix + filepath.ToSlash(rel)
		result, err := a.svc.StartTransfer(ctx, core.TransferRequest{Direction: "upload", Object: core.ObjectRef{ProfileID: target.ProfileID, Bucket: target.Bucket, Key: key}, LocalPath: filename})
		if err != nil {
			return err
		}
		if result.Plan != nil {
			if !yes {
				plans = append(plans, *result.Plan)
				return nil
			}
			plan, err := a.svc.ExecutePlan(ctx, result.Plan.ID, true)
			if err != nil {
				return err
			}
			if len(plan.Results) != 1 || !strings.HasPrefix(plan.Results[0].Status, "task:") {
				return core.NewError("internal", "The upload plan did not create a transfer")
			}
			task, err := a.svc.DB.Task(ctx, strings.TrimPrefix(plan.Results[0].Status, "task:"))
			if err != nil {
				return err
			}
			results = append(results, task)
			return nil
		}
		if result.Task != nil {
			results = append(results, *result.Task)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(plans) > 0 {
		for _, plan := range plans {
			if err := a.print(plan); err != nil {
				return nil, err
			}
		}
		return nil, core.NewError("confirmation_required", "Overwrites require review; rerun with --yes to execute the listed plans")
	}
	for i := range results {
		task, err := a.waitTask(ctx, results[i].ID)
		if err != nil {
			return nil, err
		}
		results[i] = task
	}
	return results, nil
}

func (a *app) waitTask(ctx context.Context, id string) (core.TransferTask, error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		task, err := a.svc.DB.Task(ctx, id)
		if err != nil {
			return task, err
		}
		switch task.Status {
		case "completed":
			return task, nil
		case "failed", "cancelled", "cancelling":
			if task.Error != nil {
				return task, task.Error
			}
			return task, core.NewError("partial", "The transfer did not complete; query its task ID")
		}
		select {
		case <-ctx.Done():
			return task, core.NewError("paused", "The CLI stopped waiting; the transfer task remains saved")
		case <-ticker.C:
		}
	}
}

func (a *app) mvCommand() *cobra.Command {
	return a.planOrExecute("mv", "move", "Move objects within one R2 connection")
}
func (a *app) rmCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "rm r2://profile/bucket/key [more...]", Short: "Delete objects after confirmation", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			refs := make([]core.ObjectRef, 0, len(args))
			for _, arg := range args {
				ref, err := a.resolveRemote(ctx, arg, true)
				if err != nil {
					return nil, err
				}
				refs = append(refs, ref)
			}
			plan, err := a.svc.CreatePlan(ctx, core.PlanRequest{Operation: "delete", Sources: refs})
			if err != nil {
				return nil, err
			}
			if !yes {
				if err := a.print(plan); err != nil {
					return nil, err
				}
				return nil, core.NewError("confirmation_required", "Review the plan and rerun with --yes to execute it")
			}
			return a.svc.ExecutePlan(ctx, plan.ID, true)
		})
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "Authorize execution of the generated deletion plan")
	return cmd
}

func (a *app) planOrExecute(use, operation, short string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: use + " r2://profile/bucket/key r2://profile/bucket/key", Short: short, Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return a.execute(cmd, func(ctx context.Context) (any, error) {
			source, err := a.resolveRemote(ctx, args[0], true)
			if err != nil {
				return nil, err
			}
			target, err := a.resolveRemote(ctx, args[1], true)
			if err != nil {
				return nil, err
			}
			plan, err := a.svc.CreatePlan(ctx, core.PlanRequest{Operation: operation, Sources: []core.ObjectRef{source}, Target: target})
			if err != nil {
				return nil, err
			}
			if !yes {
				if err := a.print(plan); err != nil {
					return nil, err
				}
				return nil, core.NewError("confirmation_required", "Review the plan and rerun with --yes to execute it")
			}
			return a.svc.ExecutePlan(ctx, plan.ID, true)
		})
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "Authorize execution of the generated plan")
	return cmd
}

func (a *app) planCommand() *cobra.Command {
	return &cobra.Command{Use: "plan ID", Short: "Show a saved operation plan", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return a.execute(cmd, func(ctx context.Context) (any, error) { return a.svc.Plan(ctx, args[0]) })
	}}
}

func (a *app) executeCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "execute ID", Short: "Execute a reviewed plan", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if !yes {
			return core.NewError("confirmation_required", "Review the plan and rerun with --yes to authorize execution")
		}
		return a.execute(cmd, func(ctx context.Context) (any, error) { return a.svc.ExecutePlan(ctx, args[0], true) })
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "Explicitly authorize execution of this plan")
	return cmd
}

func (a *app) transfersCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "transfers", Short: "Inspect and control saved transfers"}
	cmd.AddCommand(
		&cobra.Command{Use: "list", Short: "List transfer tasks", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			return a.execute(cmd, func(ctx context.Context) (any, error) { return a.svc.Tasks(ctx) })
		}},
		&cobra.Command{Use: "pause ID", Short: "Pause a transfer", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			return a.execute(cmd, func(ctx context.Context) (any, error) {
				if err := a.svc.PauseTransfer(ctx, args[0]); err != nil {
					return nil, err
				}
				return "Transfer paused.", nil
			})
		}},
		&cobra.Command{Use: "cancel ID", Short: "Cancel a transfer and clean up temporary data", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			return a.execute(cmd, func(ctx context.Context) (any, error) {
				if err := a.svc.CancelTransfer(ctx, args[0]); err != nil {
					return nil, err
				}
				return "Transfer cancelled.", nil
			})
		}},
		&cobra.Command{Use: "resume ID", Short: "Resume a paused transfer", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			return a.execute(cmd, func(ctx context.Context) (any, error) {
				task, err := a.svc.ResumeTransfer(ctx, args[0])
				if err != nil {
					return nil, err
				}
				return a.waitTask(ctx, task.ID)
			})
		}},
	)
	return cmd
}

func (a *app) skillCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "skill", Short: "Print the Bucky Agent Skill"}
	cmd.AddCommand(&cobra.Command{Use: "print", Short: "Print skill instructions and install directory guidance", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return a.execute(cmd, func(context.Context) (any, error) { return agent.SkillText() })
	}})
	return cmd
}

func (a *app) mcpCommand() *cobra.Command {
	return &cobra.Command{Use: "mcp", Short: "Run the read-only MCP server over stdio", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return agent.RunStdio(cmd.Context(), a.svc)
	}}
}
