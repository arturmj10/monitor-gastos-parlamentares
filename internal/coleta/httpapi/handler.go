package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/camara"
	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
	"github.com/go-chi/chi/v5"
)

type Servico interface {
	SincronizarDeputados(ctx context.Context) (int, error)
	ListarDeputados(ctx context.Context, filtro deputado.Filtro) ([]deputado.Deputado, error)
	BuscarDeputado(ctx context.Context, id int64) (deputado.Deputado, error)
	ObterDespesas(ctx context.Context, id int64, ano int) ([]deputado.Despesa, error)
}

type Handler struct {
	svc    Servico
	logger *slog.Logger
}

func NovoHandler(svc Servico, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) sincronizar(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.SincronizarDeputados(r.Context())
	if err != nil {
		h.erro(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, map[string]int{"sincronizados": n})
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	filtro := deputado.Filtro{
		UF:      r.URL.Query().Get("uf"),
		Partido: r.URL.Query().Get("partido"),
	}
	deputados, err := h.svc.ListarDeputados(r.Context(), filtro)
	if err != nil {
		h.erro(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, toDeputadosResp(deputados))
}

func (h *Handler) buscar(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.erro(w, deputado.ErrParametroInvalido)
		return
	}
	d, err := h.svc.BuscarDeputado(r.Context(), id)
	if err != nil {
		h.erro(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, toDeputadoResp(d))
}

func (h *Handler) despesas(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.erro(w, deputado.ErrParametroInvalido)
		return
	}
	ano := 0
	if s := r.URL.Query().Get("ano"); s != "" {
		ano, err = strconv.Atoi(s)
		if err != nil {
			h.erro(w, deputado.ErrParametroInvalido)
			return
		}
	}
	despesas, err := h.svc.ObterDespesas(r.Context(), id, ano)
	if err != nil {
		h.erro(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, toDespesasResp(despesas))
}

// erro traduz o erro para o status HTTP correto.
func (h *Handler) erro(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, deputado.ErrParametroInvalido):
		escreverErro(w, http.StatusBadRequest, "parametro invalido")
	case errors.Is(err, deputado.ErrDeputadoNaoEncontrado):
		escreverErro(w, http.StatusNotFound, "deputado nao encontrado")
	case errors.Is(err, camara.ErrIndisponivel):
		h.logger.Error("api da camara indisponivel", "erro", err.Error())
		escreverErro(w, http.StatusBadGateway, "servico externo indisponivel")
	default:
		h.logger.Error("erro interno", "erro", err.Error())
		escreverErro(w, http.StatusInternalServerError, "erro interno")
	}
}

type erroResp struct {
	Erro string `json:"erro"`
}

func escreverJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

func escreverErro(w http.ResponseWriter, status int, msg string) {
	escreverJSON(w, status, erroResp{Erro: msg})
}
