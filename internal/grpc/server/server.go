package server

import (
	"context"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/AlexMickh/speak-protos/pkg/api/user"
	"github.com/AlexMickh/speak-user/internal/domain/models"
	"github.com/AlexMickh/speak-user/pkg/sl"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service interface {
	SaveUser(
		ctx context.Context,
		email string,
		username string,
		password string,
		description string,
		image *models.Image,
	) (string, error)
	GetUser(ctx context.Context, email string) (models.User, error)
	VerifyEmail(ctx context.Context, id string) error
	UpdateUser(
		ctx context.Context,
		id string,
		username *string,
		description *string,
		image *models.Image,
	) (models.User, error)
	DeleteUser(ctx context.Context, id string) error
}

type AuthClient interface {
	GetUserId(ctx context.Context, token string) (string, error)
}

type Server struct {
	user.UnimplementedUserServer
	service    Service
	authClient AuthClient
}

func New(service Service, authClient AuthClient) *Server {
	return &Server{
		service:    service,
		authClient: authClient,
	}
}

func (s *Server) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.CreateUserResponse, error) {
	const op = "grpc.server.CreateUser"

	ctx = sl.GetFromCtx(ctx).With(ctx, slog.String("op", op))

	var image *models.Image

	if req.GetUsername() == "" {
		sl.GetFromCtx(ctx).Error(ctx, "username is empty")
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}
	if req.GetEmail() == "" {
		sl.GetFromCtx(ctx).Error(ctx, "email is empty")
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.GetPassword() == "" {
		sl.GetFromCtx(ctx).Error(ctx, "password is empty")
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	_, err := mail.ParseAddress(req.GetEmail())
	if err != nil {
		sl.GetFromCtx(ctx).Error(ctx, "email is not valid")
		return nil, status.Error(codes.InvalidArgument, "not valid email")
	}

	if req.GetProfileImage() == nil {
		image = nil
	} else {
		image = &models.Image{
			ID:   uuid.New(),
			Data: req.GetProfileImage(),
		}
	}

	id, err := s.service.SaveUser(
		ctx,
		req.GetEmail(),
		req.GetUsername(),
		req.GetPassword(),
		req.GetDescription(),
		image,
	)
	if err != nil {
		sl.GetFromCtx(ctx).Error(ctx, "failed to save user", sl.Err(err))
		return nil, status.Error(codes.Internal, "failed to save user")
	}

	return &user.CreateUserResponse{
		Id: id,
	}, nil
}

func (s *Server) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
	const op = "grpc.server.GetUser"

	ctx = sl.GetFromCtx(ctx).With(ctx, slog.String("op", op))

	if req.GetEmail() == "" {
		sl.GetFromCtx(ctx).Error(ctx, "email is empty")
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	_, err := mail.ParseAddress(req.GetEmail())
	if err != nil {
		sl.GetFromCtx(ctx).Error(ctx, "email is not valid")
		return nil, status.Error(codes.InvalidArgument, "not valid email")
	}

	userModel, err := s.service.GetUser(ctx, req.GetEmail())
	if err != nil {
		sl.GetFromCtx(ctx).Error(ctx, "failed to get user", sl.Err(err))
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &user.GetUserResponse{
		Id:              userModel.ID.String(),
		Email:           userModel.Email,
		Username:        *userModel.Username,
		Description:     *userModel.Description,
		ProfileImageUrl: *userModel.ProfileImageUrl,
		Password:        userModel.Password,
	}, nil
}

func (s *Server) VerifyEmail(ctx context.Context, req *user.VerifyEmailRequest) (*emptypb.Empty, error) {
	const op = "grpc.server.VerifyEmail"

	ctx = sl.GetFromCtx(ctx).With(ctx, slog.String("op", op))

	if req.GetId() == "" {
		sl.GetFromCtx(ctx).Error(ctx, "id is empty")
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := s.service.VerifyEmail(ctx, req.GetId())
	if err != nil {
		sl.GetFromCtx(ctx).Error(ctx, "failed to verify email", sl.Err(err))
		return nil, status.Error(codes.Internal, "failed to verify email")
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) UpdateUserInfo(
	ctx context.Context,
	req *user.UpdateUserInfoRequest,
) (*user.UpdateUserInfoResponse, error) {
	const op = "grpc.server.UpdateUserInfo"

	ctx = sl.GetFromCtx(ctx).With(ctx, slog.String("op", op))

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		sl.GetFromCtx(ctx).Error(ctx, "failed to get metadata")
		return nil, status.Error(codes.InvalidArgument, "metadata is empty")
	}

	auth, ok := md["authorization"]
	if !ok {
		sl.GetFromCtx(ctx).Error(ctx, "failed to get auth header")
		return nil, status.Error(codes.InvalidArgument, "authorization header is empty")
	}

	if strings.Split(auth[0], " ")[0] != "Bearer" {
		sl.GetFromCtx(ctx).Error(ctx, "wrong token type")
		return nil, status.Error(codes.InvalidArgument, "wrong token type, need Bearer")
	}

	token := strings.Split(auth[0], " ")[1]

	id, err := s.authClient.GetUserId(ctx, token)
	if err != nil {
		sl.GetFromCtx(ctx).Error(ctx, "failed to get user id from token")
		return nil, status.Error(codes.Internal, "failed to get user id")
	}

	var image *models.Image
	if req.ProfileImage == nil {
		image = nil
	} else {
		image = &models.Image{
			ID:   uuid.New(),
			Data: req.GetProfileImage(),
		}
	}
	var username, description *string
	if req.GetUsername() == "" {
		username = nil
	} else {
		username = &req.Username
	}
	if req.GetDescription() == "" {
		description = nil
	} else {
		description = &req.Description
	}

	userInfo, err := s.service.UpdateUser(ctx, id, username, description, image)
	if err != nil {
		sl.GetFromCtx(ctx).Error(ctx, "failed to update user", sl.Err(err))
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	empty := ""
	if userInfo.Username == nil {
		userInfo.Username = &empty
	}
	if userInfo.Description == nil {
		userInfo.Description = &empty
	}
	if userInfo.ProfileImageUrl == nil {
		userInfo.ProfileImageUrl = &empty
	}

	return &user.UpdateUserInfoResponse{
		Id:              userInfo.ID.String(),
		Email:           userInfo.Email,
		Username:        *userInfo.Username,
		Description:     *userInfo.Description,
		ProfileImageUrl: *userInfo.ProfileImageUrl,
	}, nil
}

func (s *Server) DeleteUser(ctx context.Context, req *user.DeleteUserRequest) (*emptypb.Empty, error) {
	const op = "grpc.server.DeleteUser"

	ctx = sl.GetFromCtx(ctx).With(ctx, slog.String("op", op))

	if req.GetId() == "" {
		sl.GetFromCtx(ctx).Error(ctx, "id is empty")
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := s.service.DeleteUser(ctx, req.GetId())
	if err != nil {
		sl.GetFromCtx(ctx).Error(ctx, "failed to delete user", sl.Err(err))
		return nil, status.Error(codes.Internal, "falied to delete user")
	}

	return &emptypb.Empty{}, nil
}
