package camara

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListarDeputados_Sucesso(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/deputados", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dados":[{"id":204379,"nome":"Acácio Favacho","siglaPartido":"MDB","siglaUf":"AP","urlFoto":"http://foto","email":"dep@camara.leg.br"}],
			"links":[{"rel":"self","href":"x"}]
		}`))
	}))
	defer srv.Close()

	c := NovoClient(srv.URL, 2*time.Second)
	deputados, err := c.ListarDeputados(context.Background())

	require.NoError(t, err)
	require.Len(t, deputados, 1)
	assert.Equal(t, int64(204379), deputados[0].ID)
	assert.Equal(t, "Acácio Favacho", deputados[0].Nome)
	assert.Equal(t, "MDB", deputados[0].Partido)
	assert.Equal(t, "AP", deputados[0].UF)
}

func TestBuscarDeputado_Sucesso(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/deputados/204379", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"dados":{"id":204379,"ultimoStatus":{"nome":"Acácio Favacho","siglaPartido":"MDB","siglaUf":"AP","urlFoto":"http://foto","email":null}}
		}`))
	}))
	defer srv.Close()

	c := NovoClient(srv.URL, 2*time.Second)
	d, err := c.BuscarDeputado(context.Background(), 204379)

	require.NoError(t, err)
	assert.Equal(t, int64(204379), d.ID)
	assert.Equal(t, "Acácio Favacho", d.Nome)
	assert.Equal(t, "AP", d.UF)
	assert.Equal(t, "", d.Email) // email null vira string vazia
}

func TestBuscarDeputado_NaoEncontrado(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NovoClient(srv.URL, 2*time.Second)
	_, err := c.BuscarDeputado(context.Background(), 999999999)

	assert.ErrorIs(t, err, deputado.ErrDeputadoNaoEncontrado)
}

func TestGet_Status500_Indisponivel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NovoClient(srv.URL, 2*time.Second)
	_, err := c.BuscarDeputado(context.Background(), 1)

	assert.ErrorIs(t, err, ErrIndisponivel)
}

func TestGet_Timeout_Indisponivel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"dados":{}}`))
	}))
	defer srv.Close()

	c := NovoClient(srv.URL, 10*time.Millisecond) // timeout menor que a resposta
	_, err := c.BuscarDeputado(context.Background(), 1)

	assert.ErrorIs(t, err, ErrIndisponivel)
}
