package user

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	userv1 "github.com/parthasarathi/go-grpc-http/gen/go/user/v1"
	"github.com/parthasarathi/go-grpc-http/internal/user/service"
)

type Server struct {
	userv1.UnimplementedUserServiceServer
	svc *service.UserService
}

func NewServer(svc *service.UserService) *Server {
	return &Server{svc: svc}
}

func (s *Server) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	user, err := s.svc.Register(ctx, req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "register: %v", err)
	}

	return &userv1.RegisterResponse{
		UserId:    user.ID,
		Email:     user.Email,
		AuthToken: user.AuthToken,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}, nil
}

func (s *Server) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	token, userID, expiresAt, err := s.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Errorf(codes.Internal, "login: %v", err)
	}

	return &userv1.LoginResponse{
		AccessToken: token,
		UserId:      userID,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *Server) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.svc.GetUser(ctx, req.UserId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}

	return &userv1.GetUserResponse{
		UserId:    user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}, nil
}

func (s *Server) GetAuthToken(ctx context.Context, req *userv1.GetAuthTokenRequest) (*userv1.GetAuthTokenResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.svc.GetUser(ctx, req.UserId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}

	return &userv1.GetAuthTokenResponse{
		AuthToken: user.AuthToken,
	}, nil
}

func (s *Server) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.svc.UpdateUser(ctx, req.UserId, req.FirstName, req.LastName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "update user: %v", err)
	}

	return &userv1.UpdateUserResponse{
		UserId:    user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}, nil
}
