package ceap

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// montarLinha cria uma linha de 32 colunas com os campos relevantes posicionados.
func montarLinha(ideCadastro, descricao, fornecedor, cnpjcpf, data, valor, mes, ano, url string) []string {
	l := make([]string, totalColunas)
	l[colIdeCadastro] = ideCadastro
	l[colDescricao] = descricao
	l[colFornecedor] = fornecedor
	l[colCNPJCPF] = cnpjcpf
	l[colDataEmissao] = data
	l[colValorLiq] = valor
	l[colMes] = mes
	l[colAno] = ano
	l[colURLDoc] = url
	return l
}

func montarZip(t *testing.T, linhas [][]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("Ano-2099.csv")
	require.NoError(t, err)

	cw := csv.NewWriter(f)
	cw.Comma = ';'
	for _, l := range linhas {
		require.NoError(t, cw.Write(l))
	}
	cw.Flush()
	require.NoError(t, cw.Error())
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestBaixarDespesas_MapeiaEFiltra(t *testing.T) {
	cabecalho := make([]string, totalColunas)
	valida := montarLinha("204379", "MANUTENCAO DE ESCRITORIO", "Amoretto cafes", "085.324.290/0013-1",
		"2024-02-07T00:00:00", "1148.70", "2", "2024", "http://doc")
	semID := montarLinha("", "LIDERANCA", "Fornecedor X", "000", "2024-01-01T00:00:00", "500", "1", "2024", "")

	zipBytes := montarZip(t, [][]string{cabecalho, valida, semID})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(zipBytes)
	}))
	defer srv.Close()

	var recebidas []deputado.Despesa
	c := NovoClient(srv.URL, 10*time.Second)
	err := c.BaixarDespesas(context.Background(), 2099, func(lote []deputado.Despesa) error {
		recebidas = append(recebidas, lote...)
		return nil
	})

	require.NoError(t, err)
	require.Len(t, recebidas, 1) // a linha sem ideCadastro foi ignorada
	d := recebidas[0]
	assert.Equal(t, int64(204379), d.DeputadoID)
	assert.Equal(t, 2024, d.Ano)
	assert.Equal(t, 2, d.Mes)
	assert.Equal(t, "MANUTENCAO DE ESCRITORIO", d.Tipo)
	assert.Equal(t, "Amoretto cafes", d.Fornecedor)
	assert.Equal(t, 1148.70, d.Valor)
	assert.Equal(t, "2024-02-07", d.Data)
	assert.Equal(t, "http://doc", d.URLDocumento)
}

func TestBaixarDespesas_Status404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NovoClient(srv.URL, 10*time.Second)
	err := c.BaixarDespesas(context.Background(), 2099, func([]deputado.Despesa) error { return nil })
	assert.ErrorIs(t, err, ErrIndisponivel)
}

func TestLinhaParaDespesa_SemIdeCadastro(t *testing.T) {
	l := montarLinha("", "x", "y", "z", "2024-01-01", "10", "1", "2024", "")
	_, ok := linhaParaDespesa(l)
	assert.False(t, ok)
}

func TestLinhaParaDespesa_ColunasInsuficientes(t *testing.T) {
	_, ok := linhaParaDespesa([]string{"a", "b", "204379"})
	assert.False(t, ok)
}

func TestParseValor(t *testing.T) {
	assert.Equal(t, 1148.70, parseValor("1148.70"))
	assert.Equal(t, 1148.70, parseValor("1148,70")) // fallback vírgula
	assert.Equal(t, 0.0, parseValor(""))
}
