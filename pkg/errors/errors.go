package errors

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPCError converts domain errors to appropriate gRPC status codes.
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	switch {
	case strings.Contains(msg, "not found"):
		return status.Error(codes.NotFound, msg)
	case strings.Contains(msg, "already registered"),
		strings.Contains(msg, "already exists"):
		return status.Error(codes.AlreadyExists, msg)
	case strings.Contains(msg, "invalid credentials"):
		return status.Error(codes.Unauthenticated, msg)
	case strings.Contains(msg, "insufficient"),
		strings.Contains(msg, "can only"):
		return status.Error(codes.FailedPrecondition, msg)
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}
