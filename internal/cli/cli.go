package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/amigoer/bucky/internal/agent"
	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/service"
	"github.com/spf13/cobra"
)

type app struct {
	svc    *service.Service
	json   bool
	stdout io.Writer
}

type output struct {
	Data  any         `json:"data,omitempty"`
	Error *core.Error `json:"error,omitempty"`
}

func Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	a := &app{stdout: os.Stdout}
	root := &cobra.Command{
		Use:           "bucky",
		Short:         "Browse and manage object storage",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == "help" || cmd.Name() == "completion" {
				return nil
			}
			backend, err := service.New(ctx, service.Options{})
			if err != nil {
				return err
			}
			a.svc = backend
			return nil
		},
		PersistentPostRun: func(*cobra.Command, []string) {
			if a.svc != nil {
				_ = a.svc.Close()
			}
		},
	}
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.PersistentFlags().BoolVar(&a.json, "json", false, "Write a stable JSON result")

	root.AddCommand(a.profilesCommand(), a.lsCommand(), a.statCommand(), a.findCommand(), a.presignCommand(), a.duCommand(), a.cpCommand(), a.mvCommand(), a.rmCommand(), a.planCommand(), a.executeCommand(), a.transfersCommand(), a.skillCommand(), a.mcpCommand())

	if err := root.ExecuteContext(ctx); err != nil {
		var reported *exitError
		if errors.As(err, &reported) && reported.reported {
			return err
		}
		var safe *core.Error
		if !errors.As(err, &safe) {
			// Cobra returns plain errors for unknown commands, flags and invalid argument counts.
			return a.fail(core.Invalid("Invalid command usage; run bucky --help"))
		}
		return a.fail(err)
	}
	return nil
}

func (a *app) print(value any) error {
	if a.json {
		return json.NewEncoder(a.stdout).Encode(output{Data: value})
	}
	var writeErr error
	writef := func(format string, args ...any) {
		if writeErr == nil {
			_, writeErr = fmt.Fprintf(a.stdout, format, args...)
		}
	}
	switch v := value.(type) {
	case []core.Profile:
		if len(v) == 0 {
			writef("No connections saved.\n")
			return writeErr
		}
		for _, p := range v {
			scope := p.Bucket
			if scope == "" {
				scope = "all accessible buckets"
			}
			readonly := ""
			if p.ReadOnly {
				readonly = " (read-only)"
			}
			writef("%s\t%s\t%s\t%s%s\n", p.ID, p.Name, p.ProviderType, scope, readonly)
		}
	case []core.Bucket:
		for _, b := range v {
			writef("%s\n", b.Name)
		}
	case core.ListPage:
		for _, prefix := range v.Prefixes {
			writef("DIR\t%s\n", prefix)
		}
		for _, o := range v.Objects {
			writef("%d\t%s\t%s\n", o.Size, o.LastModified.Format("2006-01-02 15:04"), o.Key)
		}
		if v.NextCursor != "" {
			writef("next cursor: %s\n", v.NextCursor)
		}
	case core.ObjectInfo:
		writef("key\t%s\nsize\t%d\netag\t%s\nmodified\t%s\ncontent-type\t%s\n", v.Key, v.Size, v.ETag, v.LastModified.Format("2006-01-02T15:04:05Z"), v.ContentType)
	case core.ShareLink:
		writef("%s\n", v.URL)
		if !v.ExpiresAt.IsZero() {
			writef("expires\t%s\n", v.ExpiresAt.Format("2006-01-02T15:04:05Z"))
		}
	case core.Usage:
		writef("objects\t%d\nbytes\t%d\nprefix\t%s\ncomplete\t%t\n", v.Objects, v.Bytes, v.Prefix, v.Complete)
		for kind, size := range v.Types {
			writef("%s\t%d\n", kind, size)
		}
	case core.OperationPlan:
		writef("plan\t%s\noperation\t%s\nstatus\t%s\nitems\t%d\nbytes\t%d\nexpires\t%s\n", v.ID, v.Operation, v.Status, len(v.Items), v.Bytes, v.ExpiresAt.Format("2006-01-02T15:04:05Z"))
		for _, item := range v.Items {
			target := item.Source.Key
			if item.Target != nil {
				target = item.Target.Key
			}
			writef("%s\t%s\t%d\n", item.Source.Key, target, item.Source.Size)
		}
		if v.Warning != "" {
			writef("warning\t%s\n", v.Warning)
		}
		for _, result := range v.Results {
			if result.Error != nil {
				key := result.Source.Key
				if result.Target != nil {
					key = result.Target.Key
				}
				writef("failed\t%s\t%s\n", key, result.Error.Code)
			}
		}
		if v.Status == "pending" {
			writef("authorize\tbucky execute %s --yes\n", v.ID)
		}
	case core.TransferTask:
		writef("task\t%s\nstatus\t%s\ncompleted\t%d\nsize\t%d\n", v.ID, v.Status, v.Completed, v.Size)
		if v.Error != nil {
			writef("error\t%s\n", v.Error.Code)
		}
	case []core.TransferTask:
		for _, t := range v {
			writef("%s\t%s\t%s\t%d/%d\n", t.ID, t.Status, t.Object.Key, t.Completed, t.Size)
		}
	case agent.Envelope:
		if v.Error != nil {
			writef("%s: %s\n", v.Error.Code, v.Error.Message)
		} else {
			writeErr = json.NewEncoder(a.stdout).Encode(v.Data)
		}
	case string:
		writef("%s\n", v)
	default:
		writeErr = json.NewEncoder(a.stdout).Encode(v)
	}
	return writeErr
}

func (a *app) fail(err error) error {
	safe := agent.SafeError(err)
	if a.json {
		_ = json.NewEncoder(os.Stdout).Encode(output{Error: safe})
	} else {
		fmt.Fprintf(os.Stderr, "bucky: %s: %s\n", safe.Code, safe.Message)
	}
	code := 1
	switch safe.Code {
	case "invalid_argument", "configuration", "credentials_unavailable":
		code = 2
	case "partial":
		code = 3
	case "confirmation_required":
		code = 4
	}
	return &exitError{code: code, message: safe.Message}
}

type exitError struct {
	code     int
	message  string
	reported bool
}

func (e *exitError) Error() string { return e.message }

func (a *app) execute(command *cobra.Command, run func(context.Context) (any, error)) error {
	value, err := run(command.Context())
	if err != nil {
		return err
	}
	if err := a.print(value); err != nil {
		return err
	}
	if plan, ok := value.(core.OperationPlan); ok && plan.Status == "partial" {
		return &exitError{code: 3, message: "The plan completed with partial failures", reported: true}
	}
	return nil
}

func (a *app) resolveRemote(ctx context.Context, value string, object bool) (core.ObjectRef, error) {
	return agent.ResolveRemote(ctx, a.svc, value, object)
}

func (a *app) profile(ctx context.Context, value string) (string, error) {
	return agent.ResolveProfile(ctx, a.svc, value)
}

func containsSecretFlag(args []string) (string, bool) {
	for _, arg := range args {
		lower := strings.ToLower(strings.TrimLeft(arg, "-"))
		name := strings.SplitN(lower, "=", 2)[0]
		if name == "secret" || name == "access-key" || name == "secret-key" || name == "accesskeyid" || name == "secretaccesskey" || name == "session-token" {
			return arg, true
		}
	}
	return "", false
}

func validateFlags(cmd *cobra.Command) error {
	if flag, ok := containsSecretFlag(os.Args[1:]); ok && strings.HasPrefix(flag, "-") {
		return core.Invalid("Do not pass storage credentials as command line arguments; use the keychain or documented environment variables")
	}
	return nil
}

func readSecret(prompt string) (string, error) {
	return readPassword(prompt)
}

func exitCode(err error) int {
	var target *exitError
	if errors.As(err, &target) {
		return target.code
	}
	return 1
}

func Main() {
	if err := Run(context.Background()); err != nil {
		os.Exit(exitCode(err))
	}
}
