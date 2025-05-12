package app

import (
	"context"
	"log/slog"

	"github.com/AlexMickh/speak-user/internal/config"
	"github.com/AlexMickh/speak-user/internal/service"
	"github.com/AlexMickh/speak-user/internal/storage/minio"
	"github.com/AlexMickh/speak-user/internal/storage/mongo"
	"github.com/AlexMickh/speak-user/pkg/sl"
)

type App struct {
	db      *mongo.Storage
	s3      *minio.Minio
	cfg     *config.Config
	service *service.Service
}

func Register(ctx context.Context, cfg *config.Config) *App {
	const op = "app.Register"

	ctx = sl.GetFromCtx(ctx).With(ctx,
		slog.String("op", op),
	)

	sl.GetFromCtx(ctx).Info(ctx, "initing mongo db")
	db, err := mongo.New(ctx, cfg.DB)
	if err != nil {
		sl.GetFromCtx(ctx).Fatal(ctx, "failed to init mongo db", sl.Err(err))
	}

	sl.GetFromCtx(ctx).Info(ctx, "initing minio")
	minio, err := minio.New(ctx, cfg.Minio)
	if err != nil {
		sl.GetFromCtx(ctx).Fatal(ctx, "failed to init minio", sl.Err(err))
	}

	sl.GetFromCtx(ctx).Info(ctx, "initing service")
	service := service.New(db, minio)

	return &App{
		db:      db,
		s3:      minio,
		cfg:     cfg,
		service: service,
	}
}

func (a *App) GracefulStop(ctx context.Context) {
	a.db.Close(ctx)
}
