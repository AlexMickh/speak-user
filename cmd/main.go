package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/AlexMickh/speak-user/internal/config"
	"github.com/AlexMickh/speak-user/pkg/sl"
)

func main() {
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = sl.New(ctx, os.Stdout, cfg.Env)

	sl.GetFromCtx(ctx).Info(ctx, "logger is working", slog.String("env", cfg.Env))
}
