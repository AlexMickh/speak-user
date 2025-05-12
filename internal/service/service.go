package service

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexMickh/speak-user/internal/domain/models"
	"github.com/google/uuid"
)

//go:generate mockgen -destination mocks/db_mock.go github.com/AlexMickh/speak-user/internal/service DB
type DB interface {
	SaveUser(ctx context.Context, user models.User) error
	GetUser(ctx context.Context, email string) (models.User, error)
	ChangeEmailVerified(ctx context.Context, id uuid.UUID) error
	UpdateUser(
		ctx context.Context,
		id uuid.UUID,
		username string,
		description string,
		profileImageUrl string,
	) error
	DeleteUser(ctx context.Context, id uuid.UUID) (string, error)
}

//go:generate mockgen -destination mocks/s3_mock.go github.com/AlexMickh/speak-user/internal/service S3
type S3 interface {
	SaveImage(ctx context.Context, image *models.Image) (string, error)
	GetImageUrl(ctx context.Context, imageId string) (string, error)
	DeleteImage(ctx context.Context, imageId string) error
}

type Service struct {
	db DB
	s3 S3
}

func New(db DB, s3 S3) *Service {
	return &Service{
		db: db,
		s3: s3,
	}
}

func (s *Service) SaveUser(
	ctx context.Context,
	email string,
	username string,
	password string,
	description string,
	image *models.Image,
) (string, error) {
	const op = "service.SaveUser"

	id := uuid.New()

	profileImageUrl, err := s.s3.SaveImage(ctx, image)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	user := models.User{
		ID:              id,
		Email:           email,
		Username:        username,
		Password:        password,
		Description:     description,
		ProfileImageUrl: profileImageUrl,
		IsEmailVerified: false,
		CreatedAt:       time.Now().Unix(),
		UpdatedAt:       time.Now().Unix(),
	}

	err = s.db.SaveUser(ctx, user)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id.String(), nil
}

func (s *Service) GetUser(ctx context.Context, email string) (models.User, error) {
	const op = "service.GetUser"

	user, err := s.db.GetUser(ctx, email)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Service) VerifyEmail(ctx context.Context, id string) error {
	const op = "service.VerifyEmail"

	uuid, err := uuid.FromBytes([]byte(id))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = s.db.ChangeEmailVerified(ctx, uuid)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Service) UpdateUser(
	ctx context.Context,
	id string,
	username string,
	description string,
	image *models.Image,
) error {
	const op = "serive.UpdateUser"

	uuid, err := uuid.FromBytes([]byte(id))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if image != nil {
		url, err := s.s3.SaveImage(ctx, image)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		err = s.db.UpdateUser(ctx, uuid, username, description, url)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	err = s.db.UpdateUser(ctx, uuid, username, description, "")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Service) DeleteUser(ctx context.Context, id string) error {
	const op = "service.DeleteUser"

	uuid, err := uuid.FromBytes([]byte(id))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	profileImageUrl, err := s.db.DeleteUser(ctx, uuid)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = s.s3.DeleteImage(ctx, profileImageUrl)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
