package s3compat

import (
	"context"
	"io"
	"maps"
	"math"
	"slices"

	"github.com/amigoer/bucky/internal/core"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	minPartSize = int64(5 << 20)
	maxPartSize = int64(5 << 30)
	maxParts    = 10000
)

func (s *Store) CreateMultipart(ctx context.Context, ref core.ObjectRef, opts core.PutOptions) (string, error) {
	if err := s.validateRef(ref); err != nil {
		return "", err
	}
	if err := validatePutOptions(opts); err != nil {
		return "", err
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	out, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{Bucket: new(ref.Bucket), Key: new(ref.Key), ContentType: optionalString(opts.ContentType), Metadata: maps.Clone(opts.Metadata)}, singleAttempt)
	if err != nil {
		return "", translateError(err)
	}
	if aws.ToString(out.UploadId) == "" {
		return "", protocolError()
	}
	return aws.ToString(out.UploadId), nil
}

func (s *Store) UploadPart(ctx context.Context, ref core.ObjectRef, uploadID string, number int32, body io.Reader, size int64) (core.Part, error) {
	if err := s.validateUpload(ref, uploadID); err != nil {
		return core.Part{}, err
	}
	if number < 1 || number > maxParts || body == nil || size < 0 || size > maxPartSize {
		return core.Part{}, core.Invalid("Upload part requires a number from 1 to 10000 and a body of at most 5 GiB")
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	out, err := s.client.UploadPart(ctx, &s3.UploadPartInput{Bucket: new(ref.Bucket), Key: new(ref.Key), UploadId: new(uploadID), PartNumber: new(number), Body: body, ContentLength: new(size)}, uploadAttempts(body), unsignedPayload)
	if err != nil {
		return core.Part{}, translateError(err)
	}
	if aws.ToString(out.ETag) == "" {
		return core.Part{}, protocolError()
	}
	return core.Part{Number: number, ETag: aws.ToString(out.ETag), Size: size, SHA256: aws.ToString(out.ChecksumSHA256)}, nil
}

func (s *Store) ListParts(ctx context.Context, ref core.ObjectRef, uploadID string) ([]core.Part, error) {
	if err := s.validateUpload(ref, uploadID); err != nil {
		return nil, err
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	input := &s3.ListPartsInput{Bucket: new(ref.Bucket), Key: new(ref.Key), UploadId: new(uploadID), MaxParts: new(int32(1000))}
	parts := []core.Part{}
	seenMarkers := map[string]bool{}
	seenParts := map[int32]bool{}
	for {
		out, err := s.client.ListParts(ctx, input)
		if err != nil {
			return nil, translateError(err)
		}
		for _, part := range out.Parts {
			number := aws.ToInt32(part.PartNumber)
			if number < 1 || number > maxParts || seenParts[number] || aws.ToString(part.ETag) == "" || aws.ToInt64(part.Size) < 0 || aws.ToInt64(part.Size) > maxPartSize {
				return nil, protocolError()
			}
			seenParts[number] = true
			parts = append(parts, core.Part{Number: number, ETag: aws.ToString(part.ETag), Size: aws.ToInt64(part.Size), SHA256: aws.ToString(part.ChecksumSHA256)})
		}
		if !aws.ToBool(out.IsTruncated) {
			slices.SortFunc(parts, func(a, b core.Part) int { return int(a.Number - b.Number) })
			return parts, nil
		}
		marker := aws.ToString(out.NextPartNumberMarker)
		if marker == "" || seenMarkers[marker] {
			return nil, protocolError()
		}
		seenMarkers[marker] = true
		input.PartNumberMarker = new(marker)
	}
}

func (s *Store) CompleteMultipart(ctx context.Context, ref core.ObjectRef, uploadID string, parts []core.Part, opts core.PutOptions) (core.ObjectInfo, error) {
	if err := s.validateUpload(ref, uploadID); err != nil {
		return core.ObjectInfo{}, err
	}
	if err := validatePutOptions(opts); err != nil {
		return core.ObjectInfo{}, err
	}
	if len(parts) == 0 || len(parts) > maxParts {
		return core.ObjectInfo{}, core.Invalid("Multipart completion requires between 1 and 10000 parts")
	}
	parts = slices.Clone(parts)
	slices.SortFunc(parts, func(a, b core.Part) int { return int(a.Number - b.Number) })
	completed := make([]types.CompletedPart, 0, len(parts))
	var size int64
	for i, part := range parts {
		if part.Number < 1 || part.Number > maxParts || part.ETag == "" || part.Size < 0 || part.Size > maxPartSize || (i > 0 && parts[i-1].Number == part.Number) || (i < len(parts)-1 && part.Size < minPartSize) || size > math.MaxInt64-part.Size {
			return core.ObjectInfo{}, core.Invalid("Multipart parts have invalid sizes, numbers, or ETags")
		}
		size += part.Size
		completed = append(completed, types.CompletedPart{PartNumber: new(part.Number), ETag: new(part.ETag)})
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	input := &s3.CompleteMultipartUploadInput{Bucket: new(ref.Bucket), Key: new(ref.Key), UploadId: new(uploadID), MultipartUpload: &types.CompletedMultipartUpload{Parts: completed}}
	if s.policy == R2Conditions {
		// R2 does not document completion conditions; the plan warns about this race.
		if err := s.checkDestination(ctx, ref, opts); err != nil {
			return core.ObjectInfo{}, err
		}
	} else {
		input.IfMatch = optionalString(opts.IfMatch)
		if opts.IfNoneMatch {
			input.IfNoneMatch = new("*")
		}
	}
	out, err := s.client.CompleteMultipartUpload(ctx, input, singleAttempt)
	if err != nil {
		return core.ObjectInfo{}, translateError(err)
	}
	if aws.ToString(out.ETag) == "" {
		return core.ObjectInfo{}, protocolError()
	}
	return core.ObjectInfo{ObjectRef: ref, Size: size, ETag: aws.ToString(out.ETag), ContentType: opts.ContentType, Metadata: maps.Clone(opts.Metadata)}, nil
}

func (s *Store) AbortMultipart(ctx context.Context, ref core.ObjectRef, uploadID string) error {
	if err := s.validateUpload(ref, uploadID); err != nil {
		return err
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	_, err := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{Bucket: new(ref.Bucket), Key: new(ref.Key), UploadId: new(uploadID)})
	if err != nil {
		return translateError(err)
	}
	return nil
}

func (s *Store) validateUpload(ref core.ObjectRef, uploadID string) error {
	if err := s.validateRef(ref); err != nil {
		return err
	}
	if uploadID == "" {
		return core.Invalid("Multipart upload ID is required")
	}
	return nil
}
