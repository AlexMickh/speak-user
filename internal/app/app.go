package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/AlexMickh/speak-protos/pkg/api/user"
	"github.com/AlexMickh/speak-user/internal/config"
	authclient "github.com/AlexMickh/speak-user/internal/grpc/clients/auth"
	"github.com/AlexMickh/speak-user/internal/grpc/server"
	"github.com/AlexMickh/speak-user/internal/service"
	"github.com/AlexMickh/speak-user/internal/storage/minio"
	"github.com/AlexMickh/speak-user/internal/storage/mongo"
	"github.com/AlexMickh/speak-user/pkg/sl"
	"google.golang.org/grpc"
)

type App struct {
	db         *mongo.Storage
	cfg        *config.Config
	server     *grpc.Server
	authClient *authclient.AuthClient
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

	sl.GetFromCtx(ctx).Info(ctx, "initing auth client")
	authClient, err := authclient.New(cfg.AuthServiceAddr)
	if err != nil {
		sl.GetFromCtx(ctx).Fatal(ctx, "failed to init auth client", sl.Err(err))
	}

	sl.GetFromCtx(ctx).Info(ctx, "initing server")
	srv := server.New(service, authClient)
	server := grpc.NewServer(grpc.UnaryInterceptor(sl.Interceptor(ctx)))
	user.RegisterUserServer(server, srv)

	return &App{
		db:         db,
		cfg:        cfg,
		server:     server,
		authClient: authClient,
	}
}

func (a *App) Run(ctx context.Context) {
	const op = "app.Run"

	sl.GetFromCtx(ctx).Info(ctx, "starting app")

	ctx = sl.GetFromCtx(ctx).With(ctx, slog.String("op", op))

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", a.cfg.Port))
	if err != nil {
		sl.GetFromCtx(ctx).Fatal(ctx, "failed to listen", sl.Err(err))
	}

	go func() {
		if err := a.server.Serve(lis); err != nil {
			sl.GetFromCtx(ctx).Fatal(ctx, "failed to listen", sl.Err(err))
		}
	}()

	sl.GetFromCtx(ctx).Info(ctx, "server started", slog.Int("port", a.cfg.Port))
}

func (a *App) GracefulStop(ctx context.Context) {
	a.server.GracefulStop()
	a.authClient.Close()
	a.db.Close(ctx)
}
