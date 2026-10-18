package core

import (
	"context"
	"io"
	"time"
)

type Profile struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ProviderType     string    `json:"providerType"`
	AccountID        string    `json:"accountId"`
	Jurisdiction     string    `json:"jurisdiction"`
	Bucket           string    `json:"bucket"`
	PublicURL        string    `json:"publicUrl"`
	ReadOnly         bool      `json:"readOnly"`
	CredentialSource string    `json:"credentialSource"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type Bucket struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type ObjectRef struct {
	ProfileID string `json:"profileId"`
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
}

type ObjectInfo struct {
	ObjectRef
	Size         int64             `json:"size"`
	ETag         string            `json:"etag"`
	LastModified time.Time         `json:"lastModified"`
	ContentType  string            `json:"contentType"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type ListRequest struct {
	ProfileID string `json:"profileId"`
	Bucket    string `json:"bucket"`
	Prefix    string `json:"prefix"`
	Delimiter string `json:"delimiter"`
	Cursor    string `json:"cursor"`
	Limit     int32  `json:"limit"`
}

type ListPage struct {
	Objects    []ObjectInfo `json:"objects"`
	Prefixes   []string     `json:"prefixes"`
	NextCursor string       `json:"nextCursor"`
}

type ReadOptions struct {
	Offset  int64
	Length  int64
	IfMatch string
}

type ObjectReader struct {
	Body io.ReadCloser
	Info ObjectInfo
}

type PutOptions struct {
	ContentType string            `json:"contentType"`
	IfNoneMatch bool              `json:"ifNoneMatch"`
	IfMatch     string            `json:"ifMatch"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ItemResult struct {
	Source ObjectRef  `json:"source"`
	Target *ObjectRef `json:"target,omitempty"`
	Status string     `json:"status"`
	Error  *Error     `json:"error,omitempty"`
}

type ShareLink struct {
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expiresAt"`
	Public    bool              `json:"public"`
}

type TextPreview struct {
	Text        string `json:"text"`
	ContentType string `json:"contentType"`
	Truncated   bool   `json:"truncated"`
}

type Part struct {
	Number int32  `json:"number"`
	ETag   string `json:"etag"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"`
}

type Capabilities struct {
	Buckets   bool `json:"buckets"`
	Presign   bool `json:"presign"`
	Multipart bool `json:"multipart"`
	Copy      bool `json:"copy"`
}

type ObjectStore interface {
	List(context.Context, ListRequest) (ListPage, error)
	Stat(context.Context, ObjectRef) (ObjectInfo, error)
	Read(context.Context, ObjectRef, ReadOptions) (*ObjectReader, error)
	Put(context.Context, ObjectRef, io.Reader, int64, PutOptions) (ObjectInfo, error)
	Delete(context.Context, []ObjectInfo) ([]ItemResult, error)
	Capabilities() Capabilities
}

type BucketLister interface {
	Buckets(context.Context) ([]Bucket, error)
}

type Presigner interface {
	Presign(context.Context, ObjectRef, time.Duration) (ShareLink, error)
}

type MultipartStore interface {
	CreateMultipart(context.Context, ObjectRef, PutOptions) (string, error)
	UploadPart(context.Context, ObjectRef, string, int32, io.Reader, int64) (Part, error)
	ListParts(context.Context, ObjectRef, string) ([]Part, error)
	CompleteMultipart(context.Context, ObjectRef, string, []Part, PutOptions) (ObjectInfo, error)
	AbortMultipart(context.Context, ObjectRef, string) error
}

type CopyStore interface {
	Copy(context.Context, ObjectInfo, ObjectRef, PutOptions) (ObjectInfo, error)
}

type Factory func(context.Context, Profile, Credentials) (ObjectStore, error)

type TransferTask struct {
	ID             string     `json:"id"`
	Direction      string     `json:"direction"`
	Object         ObjectRef  `json:"object"`
	LocalPath      string     `json:"localPath"`
	Status         string     `json:"status"`
	Size           int64      `json:"size"`
	Completed      int64      `json:"completed"`
	BytesPerSecond int64      `json:"bytesPerSecond"`
	UploadID       string     `json:"uploadId,omitempty"`
	PutState       string     `json:"putState,omitempty"`
	PartSize       int64      `json:"partSize"`
	SourceModTime  int64      `json:"sourceModTime"`
	SourceSHA256   string     `json:"sourceSha256,omitempty"`
	RemoteETag     string     `json:"remoteEtag,omitempty"`
	IfMatch        string     `json:"ifMatch,omitempty"`
	Overwrite      bool       `json:"overwrite"`
	PutOptions     PutOptions `json:"putOptions"`
	Verification   string     `json:"verification"`
	Error          *Error     `json:"error,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type TransferRequest struct {
	Direction      string    `json:"direction"`
	Object         ObjectRef `json:"object"`
	LocalPath      string    `json:"localPath"`
	ExpectedSHA256 string    `json:"expectedSha256,omitempty"`
}

type PlanRequest struct {
	Operation string      `json:"operation"`
	Sources   []ObjectRef `json:"sources"`
	Target    ObjectRef   `json:"target"`
	LocalPath string      `json:"localPath"`
}

type PlanItem struct {
	Source      ObjectInfo  `json:"source"`
	Target      *ObjectRef  `json:"target,omitempty"`
	Existing    *ObjectInfo `json:"existing,omitempty"`
	LocalPath   string      `json:"localPath,omitempty"`
	LocalSize   int64       `json:"localSize,omitempty"`
	LocalMtime  int64       `json:"localMtime,omitempty"`
	LocalSHA256 string      `json:"localSha256,omitempty"`
}

type OperationPlan struct {
	ID        string       `json:"id"`
	Operation string       `json:"operation"`
	Items     []PlanItem   `json:"items"`
	Bytes     int64        `json:"bytes"`
	Status    string       `json:"status"`
	Warning   string       `json:"warning"`
	Results   []ItemResult `json:"results"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

type Usage struct {
	ProfileID string           `json:"profileId"`
	Bucket    string           `json:"bucket"`
	Prefix    string           `json:"prefix"`
	Objects   int64            `json:"objects"`
	Bytes     int64            `json:"bytes"`
	Types     map[string]int64 `json:"types"`
	ScannedAt time.Time        `json:"scannedAt"`
	Complete  bool             `json:"complete"`
}

type FindRequest struct {
	ProfileID string `json:"profileId"`
	Bucket    string `json:"bucket"`
	Prefix    string `json:"prefix"`
	Contains  string `json:"contains"`
	Before    string `json:"before"`
	MinSize   int64  `json:"minSize"`
	MaxSize   int64  `json:"maxSize"`
	Limit     int    `json:"limit"`
	Cursor    string `json:"cursor"`
}

type FindResult struct {
	Objects    []ObjectInfo `json:"objects"`
	Scanned    int64        `json:"scanned"`
	Complete   bool         `json:"complete"`
	NextCursor string       `json:"nextCursor"`
}
