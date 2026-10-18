package agent

import (
	"context"
	"net/url"
	"strings"

	"github.com/amigoer/bucky/internal/core"
)

type Backend interface {
	Profiles(context.Context) ([]core.Profile, error)
	Buckets(context.Context, string) ([]core.Bucket, error)
	List(context.Context, core.ListRequest) (core.ListPage, error)
	Stat(context.Context, core.ObjectRef) (core.ObjectInfo, error)
	Find(context.Context, core.FindRequest) (core.FindResult, error)
	ReadText(context.Context, core.ObjectRef, int64) (core.TextPreview, error)
	Share(context.Context, core.ObjectRef, int64, bool) (core.ShareLink, error)
	Usage(context.Context, string, string, string) (core.Usage, error)
	Tasks(context.Context) ([]core.TransferTask, error)
	Plan(context.Context, string) (core.OperationPlan, error)
	CreatePlan(context.Context, core.PlanRequest) (core.OperationPlan, error)
}

type Envelope struct {
	Data  any         `json:"data"`
	Error *core.Error `json:"error"`
}

func SafeError(err error) *core.Error {
	if err == nil {
		return nil
	}
	source := core.AsError(err)
	messages := map[string]string{
		"invalid_argument":        "Invalid arguments; check the command help or tool schema",
		"confirmation_required":   "Review the saved plan, then authorize execution with bucky execute <id> --yes",
		"not_found":               "The requested object or local record was not found",
		"forbidden":               "This connection cannot access the requested bucket",
		"read_only":               "This connection is read-only",
		"unsupported":             "This operation is not supported",
		"configuration":           "Check the local connection configuration",
		"credentials_unavailable": "Credentials are unavailable; use the system keychain or the documented process environment",
		"conflict":                "The source, destination or local state changed; inspect it and create a new plan",
		"expired":                 "The plan expired; create and review a new plan",
		"storage":                 "The local database operation failed",
		"local_io":                "The local file operation failed",
		"timeout":                 "The operation timed out",
		"cancelled":               "The operation was cancelled",
		"paused":                  "The transfer is paused; inspect or resume it using its task ID",
		"partial":                 "Some operations did not complete; inspect the returned tasks and plans",
		"verification_failed":     "Object verification failed; inspect the transfer before retrying",
		"protocol":                "The storage service returned an invalid response",
		"unauthorized":            "The storage service rejected the credentials",
		"authentication":          "The storage service rejected the credentials",
		"permission_denied":       "The storage service denied this operation",
		"busy":                    "Another process owns the write lock; retry after it exits",
		"locked":                  "Another process owns the write lock; retry after it exits",
		"internal":                "The operation failed; check the local configuration and retry",
	}
	message, ok := messages[source.Code]
	if !ok {
		return core.NewError("internal", messages["internal"])
	}
	return &core.Error{Code: source.Code, Message: message, Retryable: source.Retryable}
}

func ParseRemote(value string) (core.ObjectRef, error) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "r2" || u.Opaque != "" || u.Host == "" || u.User != nil || u.Port() != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(value, "#") {
		return core.ObjectRef{}, core.Invalid("Use r2://profile/bucket/key and percent-encode URL-special key characters")
	}
	if strings.ContainsAny(u.Host, "/\\:@\x00\r\n") {
		return core.ObjectRef{}, core.Invalid("Invalid remote profile")
	}
	rawBucket, rawKey, _ := strings.Cut(strings.TrimPrefix(u.EscapedPath(), "/"), "/")
	bucket, bucketErr := url.PathUnescape(rawBucket)
	key, keyErr := url.PathUnescape(rawKey)
	if bucketErr != nil || keyErr != nil || bucket == "" || strings.ContainsAny(bucket, "/\\\x00\r\n") || len(bucket) > 63 || strings.ContainsRune(key, '\x00') || len(key) > 1024 {
		return core.ObjectRef{}, core.Invalid("A valid bucket and an object key of at most 1024 bytes are required")
	}
	return core.ObjectRef{ProfileID: u.Host, Bucket: bucket, Key: key}, nil
}

func ResolveProfile(ctx context.Context, backend Backend, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", core.Invalid("A profile ID or name is required")
	}
	profiles, err := backend.Profiles(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range profiles {
		if p.ID == value {
			return p.ID, nil
		}
	}
	match := ""
	for _, p := range profiles {
		if p.Name == value {
			if match != "" {
				return "", core.Invalid("The profile name is ambiguous; use its ID")
			}
			match = p.ID
		}
	}
	if match == "" {
		return "", core.NewError("not_found", "The connection was not found")
	}
	return match, nil
}

func ResolveRemote(ctx context.Context, backend Backend, value string, object bool) (core.ObjectRef, error) {
	ref, err := ParseRemote(value)
	if err != nil {
		return ref, err
	}
	if object && ref.Key == "" {
		return ref, core.Invalid("An explicit object key is required")
	}
	ref.ProfileID, err = ResolveProfile(ctx, backend, ref.ProfileID)
	return ref, err
}
