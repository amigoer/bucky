package s3compat

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/amigoer/bucky/internal/core"
	"github.com/aws/smithy-go"
)

func translateError(err error) *core.Error {
	if err == nil {
		return nil
	}
	if result, ok := errors.AsType[*core.Error](err); ok {
		return result
	}
	result := core.NewError("provider_error", "Storage request failed")
	var request interface{ ServiceRequestID() string }
	if errors.As(err, &request) {
		result.RequestID = safeToken(request.ServiceRequestID(), 256)
	}
	var providerCode string
	if api, ok := errors.AsType[smithy.APIError](err); ok {
		providerCode = api.ErrorCode()
		result.Provider = safeToken(providerCode, 128)
	}
	if errors.Is(err, context.Canceled) {
		result.Code, result.Message = "cancelled", "Operation cancelled"
		return result
	}
	if errors.Is(err, context.DeadlineExceeded) {
		result.Code, result.Message, result.Retryable = "timeout", "Storage request timed out", true
		return result
	}
	switch providerCode {
	case "NoSuchKey", "NoSuchBucket", "NoSuchUpload", "NotFound":
		result.Code, result.Message = "not_found", "The requested storage resource does not exist"
	case "AccessDenied", "Forbidden", "AllAccessDisabled":
		result.Code, result.Message = "permission_denied", "The credentials do not permit this storage operation"
	case "InvalidAccessKeyId", "SignatureDoesNotMatch", "InvalidToken", "ExpiredToken", "TokenRefreshRequired", "InvalidSecurity":
		result.Code, result.Message = "credentials_unavailable", "Storage credentials were rejected or have expired"
	case "PreconditionFailed", "ConditionalRequestConflict", "OperationAborted", "Conflict", "BucketAlreadyExists", "BucketAlreadyOwnedByYou":
		result.Code, result.Message = "conflict", "The object state changed or the destination already exists"
	case "NotImplemented", "UnsupportedOperation", "UnsupportedHeader":
		result.Code, result.Message = "unsupported", "The provider does not support this storage operation or condition"
	case "SlowDown", "Throttling", "ThrottlingException", "TooManyRequests", "RequestLimitExceeded":
		result.Code, result.Message, result.Retryable = "rate_limited", "Storage requests are being rate limited", true
	case "RequestTimeout", "RequestTimeoutException":
		result.Code, result.Message, result.Retryable = "timeout", "Storage request timed out", true
	case "InternalError", "InternalFailure", "ServiceUnavailable":
		result.Code, result.Message, result.Retryable = "unavailable", "Storage service is temporarily unavailable", true
	case "InvalidArgument", "InvalidRequest", "InvalidRange", "EntityTooSmall", "EntityTooLarge", "InvalidPart", "InvalidPartOrder", "MalformedXML":
		result.Code, result.Message = "invalid_argument", "The storage request contains an invalid parameter"
	default:
		var status interface{ HTTPStatusCode() int }
		if errors.As(err, &status) {
			switch code := status.HTTPStatusCode(); {
			case code == 401:
				result.Code, result.Message = "credentials_unavailable", "Storage credentials were rejected"
			case code == 403:
				result.Code, result.Message = "permission_denied", "The credentials do not permit this storage operation"
			case code == 404:
				result.Code, result.Message = "not_found", "The requested storage resource does not exist"
			case code == 409 || code == 412:
				result.Code, result.Message = "conflict", "The object state changed or the destination already exists"
			case code == 429:
				result.Code, result.Message, result.Retryable = "rate_limited", "Storage requests are being rate limited", true
			case code == 501:
				result.Code, result.Message = "unsupported", "The provider does not support this storage operation or condition"
			case code >= 500:
				result.Code, result.Message, result.Retryable = "unavailable", "Storage service is temporarily unavailable", true
			case code == 400 || code == 416:
				result.Code, result.Message = "invalid_argument", "The storage request contains an invalid parameter"
			}
		} else if network, ok := errors.AsType[net.Error](err); ok {
			result.Code, result.Message, result.Retryable = "network", "The storage service could not be reached", true
			if network.Timeout() {
				result.Code, result.Message = "timeout", "Storage request timed out"
			}
		}
	}
	return result
}

func safeToken(value string, limit int) string {
	if len(value) > limit {
		return ""
	}
	for _, c := range value {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && !strings.ContainsRune("-_.:/+=", c) {
			return ""
		}
	}
	return value
}

func protocolError() *core.Error {
	return core.NewError("provider_error", "The storage provider returned an invalid response")
}
