package s3compat

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"strings"
	"time"

	"github.com/amigoer/bucky/internal/core"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (s *Store) Copy(ctx context.Context, source core.ObjectInfo, target core.ObjectRef, opts core.PutOptions) (core.ObjectInfo, error) {
	if err := s.validateRef(source.ObjectRef); err != nil {
		return core.ObjectInfo{}, err
	}
	if err := s.validateRef(target); err != nil {
		return core.ObjectInfo{}, err
	}
	if err := validatePutOptions(opts); err != nil {
		return core.ObjectInfo{}, err
	}
	if source.ObjectRef == target {
		return core.ObjectInfo{}, core.Invalid("Copy source and destination must differ")
	}
	if source.ETag == "" || source.ETag == "*" || strings.ContainsAny(source.ETag, "\r\n") || source.Size < 0 {
		return core.ObjectInfo{}, core.Invalid("Copy requires the source object's exact ETag and size")
	}
	if _, err := copyPartSize(source.Size); err != nil {
		return core.ObjectInfo{}, err
	}
	current, err := s.Stat(ctx, source.ObjectRef)
	if err != nil {
		return core.ObjectInfo{}, err
	}
	if current.ETag != source.ETag || current.Size != source.Size || (!source.LastModified.IsZero() && !current.LastModified.Equal(source.LastModified)) {
		return core.ObjectInfo{}, core.NewError("conflict", "The source object changed after it was selected")
	}
	if opts.ContentType == "" {
		opts.ContentType = current.ContentType
	}
	if opts.Metadata == nil {
		opts.Metadata = maps.Clone(current.Metadata)
	}
	if s.policy == R2Conditions {
		if err := s.checkDestination(ctx, target, opts); err != nil {
			return core.ObjectInfo{}, err
		}
	}
	if source.Size > maxPartSize {
		return s.copyMultipart(ctx, source, target, opts)
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	input := &s3.CopyObjectInput{Bucket: new(target.Bucket), Key: new(target.Key), CopySource: new(copySource(source.ObjectRef)), CopySourceIfMatch: new(source.ETag), ContentType: optionalString(opts.ContentType), Metadata: maps.Clone(opts.Metadata), MetadataDirective: types.MetadataDirectiveReplace}
	if s.policy == S3Conditions {
		input.IfMatch = optionalString(opts.IfMatch)
		if opts.IfNoneMatch {
			input.IfNoneMatch = new("*")
		}
	}
	out, err := s.client.CopyObject(ctx, input, singleAttempt)
	if err != nil {
		return core.ObjectInfo{}, translateError(err)
	}
	if out.CopyObjectResult == nil || aws.ToString(out.CopyObjectResult.ETag) == "" {
		return core.ObjectInfo{}, protocolError()
	}
	return core.ObjectInfo{ObjectRef: target, Size: source.Size, ETag: aws.ToString(out.CopyObjectResult.ETag), LastModified: aws.ToTime(out.CopyObjectResult.LastModified), ContentType: opts.ContentType, Metadata: maps.Clone(opts.Metadata)}, nil
}

func (s *Store) copyMultipart(ctx context.Context, source core.ObjectInfo, target core.ObjectRef, opts core.PutOptions) (info core.ObjectInfo, resultErr error) {
	partSize, err := copyPartSize(source.Size)
	if err != nil {
		return info, err
	}
	uploadID, err := s.CreateMultipart(ctx, target, opts)
	if err != nil {
		return info, err
	}
	defer func() {
		if resultErr == nil {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if cleanupErr := s.AbortMultipart(cleanupCtx, target, uploadID); cleanupErr != nil && translateError(cleanupErr).Code != "not_found" {
			original := *translateError(resultErr)
			original.Message += "; incomplete multipart upload cleanup also failed"
			resultErr = &original
		}
	}()
	parts := make([]core.Part, 0, (source.Size+partSize-1)/partSize)
	for offset := int64(0); offset < source.Size; offset += partSize {
		size := min(partSize, source.Size-offset)
		part, err := s.copyPart(ctx, source, target, uploadID, int32(len(parts)+1), offset, size)
		if err != nil {
			return info, err
		}
		parts = append(parts, part)
	}
	return s.CompleteMultipart(ctx, target, uploadID, parts, opts)
}

func (s *Store) copyPart(ctx context.Context, source core.ObjectInfo, target core.ObjectRef, uploadID string, number int32, offset, size int64) (core.Part, error) {
	if s.policy == R2Conditions {
		// R2 UploadPartCopy cannot condition its source; stream a conditional GET instead.
		reader, err := s.Read(ctx, source.ObjectRef, core.ReadOptions{Offset: offset, Length: size, IfMatch: source.ETag})
		if err != nil {
			return core.Part{}, err
		}
		if reader.Info.ETag != source.ETag || reader.Info.Size != source.Size {
			_ = reader.Body.Close()
			return core.Part{}, core.NewError("conflict", "The source object changed during copying")
		}
		part, uploadErr := s.UploadPart(ctx, target, uploadID, number, reader.Body, size)
		closeErr := reader.Body.Close()
		if uploadErr != nil {
			return core.Part{}, uploadErr
		}
		if closeErr != nil {
			return core.Part{}, translateError(closeErr)
		}
		return part, nil
	}
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	out, err := s.client.UploadPartCopy(ctx, &s3.UploadPartCopyInput{Bucket: new(target.Bucket), Key: new(target.Key), UploadId: new(uploadID), PartNumber: new(number), CopySource: new(copySource(source.ObjectRef)), CopySourceIfMatch: new(source.ETag), CopySourceRange: new(fmt.Sprintf("bytes=%d-%d", offset, offset+size-1))})
	if err != nil {
		return core.Part{}, translateError(err)
	}
	if out.CopyPartResult == nil || aws.ToString(out.CopyPartResult.ETag) == "" {
		return core.Part{}, protocolError()
	}
	return core.Part{Number: number, ETag: aws.ToString(out.CopyPartResult.ETag), Size: size}, nil
}

func copyPartSize(size int64) (int64, error) {
	if size < 0 || size > maxPartSize*maxParts {
		return 0, core.Invalid("Object exceeds the multipart copy size limit")
	}
	const alignment = int64(1 << 20)
	partSize := max(int64(64<<20), (size+maxParts-1)/maxParts)
	partSize = ((partSize + alignment - 1) / alignment) * alignment
	if partSize > maxPartSize {
		return 0, core.Invalid("Object exceeds the maximum multipart part size")
	}
	return partSize, nil
}

func copySource(ref core.ObjectRef) string {
	return strings.ReplaceAll(url.PathEscape(ref.Bucket+"/"+ref.Key), "+", "%2B")
}
