package minio

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/AlexMickh/speak-user/internal/config"
	"github.com/AlexMickh/speak-user/internal/domain/models"
	"github.com/AlexMickh/speak-user/pkg/utils/retry"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Minio struct {
	mc         *minio.Client
	bucketName string
}

func New(ctx context.Context, cfg config.MinioConfig) (*Minio, error) {
	const op = "storage.minio.New"

	var mc *minio.Client

	err := retry.WithDelay(5, 500*time.Millisecond, func() error {
		var err error

		mc, err = minio.New(cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.User, cfg.Password, ""),
			Secure: cfg.IsUseSsl,
		})
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		exists, err := mc.BucketExists(ctx, cfg.BucketName)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		if !exists {
			err = mc.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
			if err != nil {
				return fmt.Errorf("%s: %w", op, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Minio{
		mc:         mc,
		bucketName: cfg.BucketName,
	}, nil
}

func (m *Minio) SaveImage(ctx context.Context, image models.Image) error {
	const op = "storage.minio.SaveImage"

	reader := bytes.NewReader(image.Data)

	_, err := m.mc.PutObject(
		ctx,
		m.bucketName,
		image.ID.String(),
		reader,
		int64(len(image.Data)),
		minio.PutObjectOptions{ContentType: "image/png"},
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (m *Minio) GetImage(ctx context.Context, imageId string) (string, error) {
	const op = "storage.minio.GetImage"

	url, err := m.mc.PresignedGetObject(ctx, m.bucketName, imageId, 5*24*time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url.String(), nil
}

func (m *Minio) DeleteImage(ctx context.Context, imageId string) error {
	const op = "storage.minio.DeleteImage"

	err := m.mc.RemoveObject(ctx, m.bucketName, imageId, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
