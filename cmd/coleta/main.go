package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/camara"
	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/config"
	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/httpapi"
	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/repository"
	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/service"
	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("falha ao carregar configuracao", "erro", err)
		os.Exit(1)
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.SlogLevel()}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := storage.Abrir(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("falha ao conectar no banco", "erro", err)
		os.Exit(1)
	}

	repo := repository.NovoGorm(db)
	if err := repo.Migrar(); err != nil {
		logger.Error("falha ao migrar o banco", "erro", err)
		os.Exit(1)
	}
	logger.Info("banco conectado e migrado")

	camaraClient := camara.NovoClient(cfg.CamaraBaseURL, cfg.CamaraTimeout)
	svc := service.Novo(camaraClient, repo)
	handler := httpapi.NovoHandler(svc, logger)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(logger, handler),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("servidor iniciado", "porta", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("servidor encerrou com erro", "erro", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("encerrando servidor")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("erro ao encerrar servidor", "erro", err)
		os.Exit(1)
	}
}
