package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/camara"
	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type servicoFake struct {
	deputado  deputado.Deputado
	lista     []deputado.Deputado
	despesas  []deputado.Despesa
	sincroniz int
	err       error
}

func (s *servicoFake) SincronizarDeputados(context.Context) (int, error) {
	return s.sincroniz, s.err
}
func (s *servicoFake) ListarDeputados(context.Context, deputado.Filtro) ([]deputado.Deputado, error) {
	return s.lista, s.err
}
func (s *servicoFake) BuscarDeputado(context.Context, int64) (deputado.Deputado, error) {
	if s.err != nil {
		return deputado.Deputado{}, s.err
	}
	return s.deputado, nil
}
func (s *servicoFake) ObterDespesas(context.Context, int64, int) ([]deputado.Despesa, error) {
	return s.despesas, s.err
}

func novoServer(svc Servico) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(logger, NovoHandler(svc, logger))
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestBuscar_200(t *testing.T) {
	h := novoServer(&servicoFake{deputado: deputado.Deputado{ID: 7, Nome: "Fulano", UF: "SP"}})
	rec := get(h, "/deputados/7")

	require.Equal(t, http.StatusOK, rec.Code)
	var body deputadoResp
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, int64(7), body.ID)
	assert.Equal(t, "Fulano", body.Nome)
}

func TestBuscar_404(t *testing.T) {
	h := novoServer(&servicoFake{err: deputado.ErrDeputadoNaoEncontrado})
	rec := get(h, "/deputados/7")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBuscar_400IDInvalido(t *testing.T) {
	h := novoServer(&servicoFake{})
	rec := get(h, "/deputados/abc")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBuscar_502ApiIndisponivel(t *testing.T) {
	h := novoServer(&servicoFake{err: camara.ErrIndisponivel})
	rec := get(h, "/deputados/7")
	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestListar_200(t *testing.T) {
	h := novoServer(&servicoFake{lista: []deputado.Deputado{{ID: 1}, {ID: 2}}})
	rec := get(h, "/deputados")

	require.Equal(t, http.StatusOK, rec.Code)
	var body []deputadoResp
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Len(t, body, 2)
}

func TestSincronizar_200(t *testing.T) {
	h := novoServer(&servicoFake{sincroniz: 513})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/deputados/sync", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDespesas_502ApiIndisponivel(t *testing.T) {
	h := novoServer(&servicoFake{err: camara.ErrIndisponivel})
	rec := get(h, "/deputados/7/despesas?ano=2024")
	assert.Equal(t, http.StatusBadGateway, rec.Code)
}
