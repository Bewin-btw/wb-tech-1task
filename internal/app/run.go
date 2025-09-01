package app

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"net/http"
	"time"

	"wb-tech-1task/internal/cache"
	"wb-tech-1task/internal/config"
	"wb-tech-1task/internal/db/postgres"
	"wb-tech-1task/internal/kafka"
	"wb-tech-1task/internal/server"
	"wb-tech-1task/internal/service"
)

func Run(ctx context.Context, cfg *config.Config, logger *zap.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	repo, err := postgres.NewPostgresRepository(cfg.DatabaseURL)
	if err != nil {
		logger.Sugar().Errorf("failed to create postgres repo: %v", err)
		return err
	}
	defer repo.Close()

	c := cache.New(cfg.CacheTTL)
	defer c.Close()

	loadCtx, loadCancel := context.WithTimeout(ctx, 10*time.Second)
	orders, err := repo.GetAllOrders(loadCtx)
	loadCancel()
	if err != nil {
		logger.Sugar().Errorf("failed to load orders for cache: %v", err)
		return err
	}
	if err := c.DBBackup(orders); err != nil {
		logger.Sugar().Errorf("failed to fill cache: %v", err)
		return err
	}
	logger.Sugar().Infof("restored %d orders into cache", c.Count())

	svc := service.NewOrderService(c, repo, logger)

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroup, svc)

	router := server.NewRouter(svc, logger)
	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		logger.Sugar().Infof("http server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Sugar().Errorf("http server error: %v", err)
			return err
		}
		return nil
	})

	g.Go(func() error {
		logger.Sugar().Info("kafka consumer starting")
		if err := consumer.Run(gctx); err != nil {
			logger.Sugar().Errorf("consumer run error: %v", err)
			return err
		}
		return nil
	})

	err = g.Wait()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = consumer.Close()
	_ = repo.Close()
	c.Close()

	return err
}
