package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"wb-tech-1task/internal/app"
	"wb-tech-1task/internal/config"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		panic(err)
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(ctx, cfg, logger)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sig:
		logger.Sugar().Infof("signal received: %v, shutting down", s)
		cancel()
		select {
		case err = <-errCh:
			if err != nil {
				logger.Sugar().Errorf("app exited with error: %v", err)
			}
		case <-time.After(15 * time.Second):
			logger.Sugar().Warn("timeout waiting for app to stop")
		}
	case err = <-errCh:
		if err != nil {
			logger.Sugar().Errorf("app run returned error: %v", err)
			cancel()
		} else {
			logger.Sugar().Info("app run finished")
		}
	}
}
