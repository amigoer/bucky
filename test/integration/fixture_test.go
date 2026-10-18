//go:build integration

package integration_test

import (
	"context"
	"crypto/rand"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/amigoer/bucky/internal/drivers/s3compat"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	minioImage     = "cgr.dev/chainguard/minio@sha256:709099319add48dfb44aa596a7070d3ddd8dd04d7ae881765719aec29df9e510"
	rustfsImage    = "rustfs/rustfs:1.0.3@sha256:9ca9559533b537d10482fe923288b676ad08f92a7069d9a17327fbae3cab688b"
	localAccessKey = "bucky-local-test-only"
	localSecretKey = "bucky-local-test-only-not-for-production"
	localRegion    = "us-east-1"
)

type protocolStore interface {
	core.ObjectStore
	core.BucketLister
	core.Presigner
	core.CopyStore
	core.MultipartStore
}

type backend struct {
	name       string
	image      string
	command    []string
	env        map[string]string
	healthPath string
}

type fixture struct {
	ctx       context.Context
	store     protocolStore
	endpoint  string
	profileID string
	bucket    string
}

func localBackends() []backend {
	return []backend{
		{
			name:    "minio",
			image:   minioImage,
			command: []string{"server", "/data", "--address", ":9000", "--console-address", ":9001"},
			env: map[string]string{
				"MINIO_ROOT_USER":     localAccessKey,
				"MINIO_ROOT_PASSWORD": localSecretKey,
				"MINIO_BROWSER":       "off",
			},
			healthPath: "/minio/health/ready",
		},
		{
			name:    "rustfs",
			image:   rustfsImage,
			command: []string{"/data"},
			env: map[string]string{
				"RUSTFS_ACCESS_KEY":        localAccessKey,
				"RUSTFS_SECRET_KEY":        localSecretKey,
				"RUSTFS_ADDRESS":           ":9000",
				"RUSTFS_VOLUMES":           "/data",
				"RUSTFS_CONSOLE_ENABLE":    "false",
				"RUSTFS_OBS_LOG_DIRECTORY": "/logs",
			},
			healthPath: "/health",
		},
	}
}

func startFixture(t *testing.T, backend backend) *fixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	t.Cleanup(cancel)
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        backend.image,
			Cmd:          backend.command,
			Env:          backend.env,
			ExposedPorts: []string{"9000/tcp"},
			Tmpfs: map[string]string{
				"/data": "rw,nosuid,nodev,mode=0777,size=512m",
				"/logs": "rw,nosuid,nodev,mode=0777,size=32m",
			},
			ConfigModifier: func(cfg *container.Config) {
				cfg.User = "65532:65532"
			},
			HostConfigModifier: func(cfg *container.HostConfig) {
				cfg.PortBindings = network.PortMap{
					network.MustParsePort("9000/tcp"): []network.PortBinding{{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "0"}},
				}
			},
			WaitingFor: wait.ForHTTP(backend.healthPath).WithPort("9000/tcp").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
		Logger:  log.New(io.Discard, "", 0),
	})
	t.Cleanup(func() {
		if ctr == nil {
			return
		}
		cleanupCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if err := ctr.Terminate(cleanupCtx); err != nil {
			t.Error("Container cleanup failed; details suppressed")
		}
	})
	requireNoError(t, "start local backend", err)
	host, err := ctr.Host(ctx)
	requireNoError(t, "resolve backend host", err)
	port, err := ctr.MappedPort(ctx, "9000/tcp")
	requireNoError(t, "resolve backend port", err)
	endpoint := "http://" + net.JoinHostPort(host, port.Port())
	credentials := core.Credentials{AccessKeyID: localAccessKey, SecretAccessKey: localSecretKey}
	profileID := "integration-" + backend.name
	store, err := s3compat.New(ctx, s3compat.Config{
		Endpoint:        endpoint,
		Region:          localRegion,
		PathStyle:       true,
		Credentials:     credentials,
		ProfileID:       profileID,
		ConditionPolicy: s3compat.R2Conditions,
	})
	requireNoError(t, "create local store", err)

	client := s3.New(s3.Options{
		BaseEndpoint: new(endpoint),
		Region:       localRegion,
		UsePathStyle: true,
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: localAccessKey, SecretAccessKey: localSecretKey}, nil
		}),
		HTTPClient:                 newHTTPClient(),
		RetryMaxAttempts:           1,
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		Logger:                     logging.Nop{},
	})
	bucket := "bucky-it-" + strings.ToLower(rand.Text())
	deadline := time.Now().Add(60 * time.Second)
	for {
		_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: new(bucket)})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("create test bucket failed after the retry deadline")
		}
		select {
		case <-ctx.Done():
			t.Fatal("create test bucket timed out")
		case <-time.After(500 * time.Millisecond):
		}
	}
	t.Cleanup(func() { cleanBucket(t, client, bucket) })
	return &fixture{ctx: ctx, store: store, endpoint: endpoint, profileID: profileID, bucket: bucket}
}

func (f *fixture) factory(ctx context.Context, p core.Profile, creds core.Credentials) (core.ObjectStore, error) {
	return s3compat.New(ctx, s3compat.Config{
		Endpoint:        f.endpoint,
		Region:          localRegion,
		PathStyle:       true,
		Credentials:     creds,
		ProfileID:       p.ID,
		ConditionPolicy: s3compat.R2Conditions,
	})
}

func cleanBucket(t *testing.T, client *s3.Client, bucket string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for range 100 {
		uploads, err := client.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{Bucket: new(bucket)})
		if err != nil {
			t.Error("Test multipart cleanup listing failed; details suppressed")
			return
		}
		for _, upload := range uploads.Uploads {
			_, err := client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{Bucket: new(bucket), Key: upload.Key, UploadId: upload.UploadId})
			if err != nil {
				t.Error("Test multipart cleanup failed; details suppressed")
				return
			}
		}
		if len(uploads.Uploads) != 0 {
			continue
		}
		page, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: new(bucket)})
		if err != nil {
			t.Error("Test bucket cleanup listing failed; details suppressed")
			return
		}
		if len(page.Contents) == 0 {
			_, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: new(bucket)})
			if err != nil {
				t.Error("Test bucket cleanup failed; details suppressed")
			}
			return
		}
		for _, object := range page.Contents {
			_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: new(bucket), Key: object.Key})
			if err != nil {
				t.Error("Test object cleanup failed; details suppressed")
				return
			}
		}
	}
	t.Error("Test bucket cleanup exceeded its page limit")
}

func (f *fixture) ref(key string) core.ObjectRef {
	return core.ObjectRef{ProfileID: f.profileID, Bucket: f.bucket, Key: key}
}

func requireNoError(t *testing.T, operation string, err error) {
	t.Helper()
	if err != nil {
		// Only the sanitized error is shown: raw SDK errors and signed URLs can contain credentials.
		result := core.AsError(err)
		t.Fatalf("%s failed: %s: %s", operation, result.Code, result.Message)
	}
}

func requireErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || core.AsError(err).Code != code {
		t.Fatalf("Expected error code %s; error details suppressed", code)
	}
}

func newHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	return &http.Client{
		Transport:     transport,
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func getPresigned(t *testing.T, ctx context.Context, link core.ShareLink, byteRange string, limit int64) (int, []byte) {
	t.Helper()
	client := newHTTPClient()
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.URL, nil)
	requireNoError(t, "create presigned request", err)
	for name, value := range link.Headers {
		req.Header.Set(name, value)
	}
	if byteRange != "" {
		req.Header.Set("Range", byteRange)
	}
	response, err := client.Do(req)
	requireNoError(t, "read presigned object over HTTP", err)
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	requireNoError(t, "read presigned response body", err)
	if int64(len(body)) > limit {
		t.Fatal("Presigned response exceeded the read limit")
	}
	return response.StatusCode, body
}
