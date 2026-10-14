package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(logger *slog.Logger, h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		escreverJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/deputados", func(r chi.Router) {
		r.Get("/", h.listar)
		r.Post("/sync", h.sincronizar)
		r.Get("/{id}", h.buscar)
		r.Get("/{id}/despesas", h.despesas)
	})

	return r
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("requisicao http",
				"metodo", r.Method,
				"rota", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duracao_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
