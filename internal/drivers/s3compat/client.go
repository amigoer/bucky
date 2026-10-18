package s3compat

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
)

type ConditionPolicy uint8

const (
	S3Conditions ConditionPolicy = iota
	R2Conditions
)

type Config struct {
	Endpoint        string
	Region          string
	PathStyle       bool
	Credentials     core.Credentials
	ProfileID       string
	ConditionPolicy ConditionPolicy
	RequestTimeout  time.Duration
}

type Store struct {
	client    *s3.Client
	presigner *s3.PresignClient
	profileID string
	policy    ConditionPolicy
	timeout   time.Duration
}

var (
	_ core.ObjectStore    = (*Store)(nil)
	_ core.BucketLister   = (*Store)(nil)
	_ core.Presigner      = (*Store)(nil)
	_ core.MultipartStore = (*Store)(nil)
	_ core.CopyStore      = (*Store)(nil)
)

func New(ctx context.Context, cfg Config) (*Store, error) {
	if err := ctx.Err(); err != nil {
		return nil, translateError(err)
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.Opaque != "" {
		return nil, core.Invalid("Endpoint must be an HTTP or HTTPS origin")
	}
	if cfg.ProfileID == "" || cfg.Region == "" || strings.ContainsAny(cfg.Region, " \t\r\n/") {
		return nil, core.Invalid("Profile ID and a valid region are required")
	}
	if cfg.Credentials.AccessKeyID == "" || cfg.Credentials.SecretAccessKey == "" {
		return nil, core.NewError("credentials_unavailable", "S3 access key and secret key are required")
	}
	if cfg.ConditionPolicy != S3Conditions && cfg.ConditionPolicy != R2Conditions {
		return nil, core.Invalid("Unknown conditional request policy")
	}
	if cfg.RequestTimeout < 0 {
		return nil, core.Invalid("Request timeout must be positive")
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 5 * time.Minute
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ResponseHeaderTimeout = 30 * time.Second
	client := s3.New(s3.Options{
		BaseEndpoint: new(strings.TrimSuffix(cfg.Endpoint, "/")),
		Region:       cfg.Region,
		UsePathStyle: cfg.PathStyle,
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			if err := ctx.Err(); err != nil {
				return aws.Credentials{}, err
			}
			return aws.Credentials{AccessKeyID: cfg.Credentials.AccessKeyID, SecretAccessKey: cfg.Credentials.SecretAccessKey, SessionToken: cfg.Credentials.SessionToken, Source: "Bucky"}, nil
		}),
		HTTPClient: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}},
		Retryer: retry.NewStandard(func(o *retry.StandardOptions) {
			o.MaxAttempts = 3
			o.MaxBackoff = 2 * time.Second
		}),
		RequestChecksumCalculation:                aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation:                aws.ResponseChecksumValidationWhenRequired,
		DisableLogOutputChecksumValidationSkipped: true,
		Logger: logging.Nop{},
	})
	return &Store{client: client, presigner: s3.NewPresignClient(client), profileID: cfg.ProfileID, policy: cfg.ConditionPolicy, timeout: cfg.RequestTimeout}, nil
}

func (s *Store) Capabilities() core.Capabilities {
	return core.Capabilities{Buckets: true, Presign: true, Multipart: true, Copy: true}
}

func (s *Store) requestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.timeout)
}

func (s *Store) validateRef(ref core.ObjectRef) error {
	if ref.ProfileID != s.profileID {
		return core.Invalid("Object belongs to a different profile")
	}
	if ref.Bucket == "" || strings.ContainsAny(ref.Bucket, "/\\\r\n\x00") || strings.HasPrefix(ref.Bucket, "arn:") {
		return core.Invalid("A bucket name is required")
	}
	if ref.Key == "" || len(ref.Key) > 1024 || strings.ContainsRune(ref.Key, '\x00') {
		return core.Invalid("Object key must contain between 1 and 1024 bytes without NUL")
	}
	return nil
}

func validatePutOptions(opts core.PutOptions) error {
	if opts.IfNoneMatch && opts.IfMatch != "" {
		return core.Invalid("IfMatch and IfNoneMatch cannot be combined")
	}
	if strings.ContainsAny(opts.IfMatch+opts.ContentType, "\r\n") {
		return core.Invalid("Object headers cannot contain line breaks")
	}
	return nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return new(value)
}

// Commit requests must not be replayed after an ambiguous network failure.
func singleAttempt(opts *s3.Options) { opts.RetryMaxAttempts = 1 }

func uploadAttempts(body io.Reader) func(*s3.Options) {
	if _, ok := body.(io.Seeker); ok {
		return func(*s3.Options) {}
	}
	return singleAttempt
}
