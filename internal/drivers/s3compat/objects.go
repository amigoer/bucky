package s3compat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (s *Store) List(ctx context.Context, req core.ListRequest) (core.ListPage, error) {
	page := core.ListPage{Objects: []core.ObjectInfo{}, Prefixes: []string{}}
	if err := s.validateRef(core.ObjectRef{ProfileID: req.ProfileID, Bucket: req.Bucket, Key: "_"}); err != nil {
		return page, err
	}
	if req.Limit < 0 || req.Limit > 1000 {
		return page, core.Invalid("List limit must be between 1 and 1000, or zero for the default")
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	input := &s3.ListObjectsV2Input{Bucket: new(req.Bucket), Prefix: optionalString(req.Prefix), Delimiter: optionalString(req.Delimiter), ContinuationToken: optionalString(req.Cursor), EncodingType: types.EncodingTypeUrl}
	if req.Limit > 0 {
		input.MaxKeys = new(req.Limit)
	}
	out, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return page, translateError(err)
	}
	decode := func(value string) (string, error) {
		if out.EncodingType == types.EncodingTypeUrl {
			decoded, err := url.QueryUnescape(value)
			if err != nil {
				return "", protocolError()
			}
			return decoded, nil
		}
		return value, nil
	}
	for _, object := range out.Contents {
		key, err := decode(aws.ToString(object.Key))
		if err != nil {
			return page, err
		}
		page.Objects = append(page.Objects, core.ObjectInfo{ObjectRef: core.ObjectRef{ProfileID: s.profileID, Bucket: req.Bucket, Key: key}, Size: aws.ToInt64(object.Size), ETag: aws.ToString(object.ETag), LastModified: aws.ToTime(object.LastModified)})
	}
	for _, prefix := range out.CommonPrefixes {
		key, err := decode(aws.ToString(prefix.Prefix))
		if err != nil {
			return page, err
		}
		page.Prefixes = append(page.Prefixes, key)
	}
	if aws.ToBool(out.IsTruncated) {
		page.NextCursor = aws.ToString(out.NextContinuationToken)
		if page.NextCursor == "" || page.NextCursor == req.Cursor {
			return page, protocolError()
		}
	}
	return page, nil
}

func (s *Store) Stat(ctx context.Context, ref core.ObjectRef) (core.ObjectInfo, error) {
	if err := s.validateRef(ref); err != nil {
		return core.ObjectInfo{}, err
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: new(ref.Bucket), Key: new(ref.Key)})
	if err != nil {
		return core.ObjectInfo{}, translateError(err)
	}
	return core.ObjectInfo{ObjectRef: ref, Size: aws.ToInt64(out.ContentLength), ETag: aws.ToString(out.ETag), LastModified: aws.ToTime(out.LastModified), ContentType: aws.ToString(out.ContentType), Metadata: maps.Clone(out.Metadata)}, nil
}

func (s *Store) Read(ctx context.Context, ref core.ObjectRef, opts core.ReadOptions) (*core.ObjectReader, error) {
	if err := s.validateRef(ref); err != nil {
		return nil, err
	}
	if opts.Offset < 0 || opts.Length < 0 || (opts.Length > 0 && opts.Offset > math.MaxInt64-opts.Length) || strings.ContainsAny(opts.IfMatch, "\r\n") {
		return nil, core.Invalid("Object range or condition is invalid")
	}
	ctx, cancel := s.requestContext(ctx)
	input := &s3.GetObjectInput{Bucket: new(ref.Bucket), Key: new(ref.Key), IfMatch: optionalString(opts.IfMatch)}
	if opts.Length > 0 {
		input.Range = new(fmt.Sprintf("bytes=%d-%d", opts.Offset, opts.Offset+opts.Length-1))
	} else if opts.Offset > 0 {
		input.Range = new(fmt.Sprintf("bytes=%d-", opts.Offset))
	}
	out, err := s.client.GetObject(ctx, input)
	if err != nil {
		cancel()
		return nil, translateError(err)
	}
	size := aws.ToInt64(out.ContentLength)
	if input.Range != nil {
		before, total, ok := strings.CutLast(aws.ToString(out.ContentRange), "/")
		parsed, err := strconv.ParseInt(total, 10, 64)
		if !ok || !strings.HasPrefix(before, "bytes ") || err != nil || parsed < size {
			_ = out.Body.Close()
			cancel()
			return nil, protocolError()
		}
		size = parsed
	}
	return &core.ObjectReader{Body: &responseBody{body: out.Body, ctx: ctx, cancel: cancel}, Info: core.ObjectInfo{ObjectRef: ref, Size: size, ETag: aws.ToString(out.ETag), LastModified: aws.ToTime(out.LastModified), ContentType: aws.ToString(out.ContentType), Metadata: maps.Clone(out.Metadata)}}, nil
}

func (s *Store) Put(ctx context.Context, ref core.ObjectRef, body io.Reader, size int64, opts core.PutOptions) (core.ObjectInfo, error) {
	if err := s.validateRef(ref); err != nil {
		return core.ObjectInfo{}, err
	}
	if err := validatePutOptions(opts); err != nil {
		return core.ObjectInfo{}, err
	}
	if body == nil || size < 0 || size > maxPartSize {
		return core.ObjectInfo{}, core.Invalid("Put requires a body and a size of at most 5 GiB; use multipart for larger objects")
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	input := &s3.PutObjectInput{Bucket: new(ref.Bucket), Key: new(ref.Key), Body: body, ContentLength: new(size), ContentType: optionalString(opts.ContentType), Metadata: maps.Clone(opts.Metadata), IfMatch: optionalString(opts.IfMatch)}
	if opts.IfNoneMatch {
		input.IfNoneMatch = new("*")
	}
	out, err := s.client.PutObject(ctx, input, singleAttempt, unsignedPayload)
	if err != nil {
		return core.ObjectInfo{}, translateError(err)
	}
	if aws.ToString(out.ETag) == "" {
		return core.ObjectInfo{}, protocolError()
	}
	return core.ObjectInfo{ObjectRef: ref, Size: size, ETag: aws.ToString(out.ETag), ContentType: opts.ContentType, Metadata: maps.Clone(opts.Metadata)}, nil
}

func (s *Store) Delete(ctx context.Context, objects []core.ObjectInfo) ([]core.ItemResult, error) {
	results := make([]core.ItemResult, 0, len(objects))
	for _, object := range objects {
		result := core.ItemResult{Source: object.ObjectRef, Status: "completed"}
		err := s.deleteObject(ctx, object)
		if err != nil {
			result.Status, result.Error = "failed", translateError(err)
		}
		results = append(results, result)
	}
	if ctx.Err() != nil {
		return results, translateError(ctx.Err())
	}
	return results, nil
}

func (s *Store) deleteObject(ctx context.Context, object core.ObjectInfo) error {
	if err := s.validateRef(object.ObjectRef); err != nil {
		return err
	}
	if object.ETag == "" || object.ETag == "*" || strings.ContainsAny(object.ETag, "\r\n") {
		return core.Invalid("Deletion requires the object's exact ETag")
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	input := &s3.DeleteObjectInput{Bucket: new(object.Bucket), Key: new(object.Key)}
	if s.policy == R2Conditions {
		// R2 does not document conditional DELETE; the plan warns about this race.
		if err := s.checkSnapshot(ctx, object); err != nil {
			return err
		}
	} else {
		input.IfMatch = new(object.ETag)
	}
	_, err := s.client.DeleteObject(ctx, input, singleAttempt)
	if err != nil {
		return translateError(err)
	}
	return nil
}

func (s *Store) checkSnapshot(ctx context.Context, expected core.ObjectInfo) error {
	current, err := s.Stat(ctx, expected.ObjectRef)
	if err != nil {
		return err
	}
	if current.ETag != expected.ETag || current.Size != expected.Size || (!expected.LastModified.IsZero() && !current.LastModified.Equal(expected.LastModified)) {
		return core.NewError("conflict", "The source object changed after it was selected")
	}
	return nil
}

func (s *Store) checkDestination(ctx context.Context, ref core.ObjectRef, opts core.PutOptions) error {
	if !opts.IfNoneMatch && opts.IfMatch == "" {
		return nil
	}
	current, err := s.Stat(ctx, ref)
	if err != nil {
		if translateError(err).Code == "not_found" {
			if opts.IfNoneMatch {
				return nil
			}
			return core.NewError("conflict", "The destination object no longer exists")
		}
		return err
	}
	if opts.IfNoneMatch || current.ETag != opts.IfMatch {
		return core.NewError("conflict", "The destination object changed or already exists")
	}
	return nil
}

func (s *Store) Buckets(ctx context.Context) ([]core.Bucket, error) {
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	buckets := []core.Bucket{}
	input := &s3.ListBucketsInput{}
	seen := map[string]bool{}
	for {
		out, err := s.client.ListBuckets(ctx, input)
		if err != nil {
			return nil, translateError(err)
		}
		for _, bucket := range out.Buckets {
			buckets = append(buckets, core.Bucket{Name: aws.ToString(bucket.Name), CreatedAt: aws.ToTime(bucket.CreationDate)})
		}
		token := aws.ToString(out.ContinuationToken)
		if token == "" {
			return buckets, nil
		}
		if seen[token] {
			return nil, protocolError()
		}
		seen[token] = true
		input.ContinuationToken = new(token)
	}
}

func (s *Store) Presign(ctx context.Context, ref core.ObjectRef, expires time.Duration) (core.ShareLink, error) {
	if err := s.validateRef(ref); err != nil {
		return core.ShareLink{}, err
	}
	if expires < time.Second || expires > 7*24*time.Hour {
		return core.ShareLink{}, core.Invalid("Presigned link lifetime must be between 1 second and 7 days")
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	out, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: new(ref.Bucket), Key: new(ref.Key)}, func(opts *s3.PresignOptions) {
		opts.Expires = expires.Truncate(time.Second)
	})
	if err != nil {
		return core.ShareLink{}, translateError(err)
	}
	u, err := url.Parse(out.URL)
	if err != nil {
		return core.ShareLink{}, protocolError()
	}
	signedAt, err := time.Parse("20060102T150405Z", u.Query().Get("X-Amz-Date"))
	if err != nil {
		return core.ShareLink{}, protocolError()
	}
	headers := make(map[string]string, len(out.SignedHeader))
	for key, values := range out.SignedHeader {
		if !strings.EqualFold(key, "host") {
			headers[key] = strings.Join(values, ",")
		}
	}
	return core.ShareLink{URL: out.URL, Headers: headers, ExpiresAt: signedAt.Add(expires.Truncate(time.Second))}, nil
}

func unsignedPayload(opts *s3.Options) {
	opts.APIOptions = append(opts.APIOptions, v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware)
}

type responseBody struct {
	body   io.ReadCloser
	ctx    context.Context
	cancel context.CancelFunc
}

func (r *responseBody) Read(p []byte) (int, error) {
	n, err := r.body.Read(p)
	if err == nil || errors.Is(err, io.EOF) {
		return n, err
	}
	if r.ctx.Err() != nil {
		return n, translateError(r.ctx.Err())
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return n, &core.Error{Code: "network", Message: "The storage response ended before the object was received", Retryable: true}
	}
	return n, translateError(err)
}

func (r *responseBody) Close() error {
	err := r.body.Close()
	r.cancel()
	if err != nil {
		return translateError(err)
	}
	return nil
}
