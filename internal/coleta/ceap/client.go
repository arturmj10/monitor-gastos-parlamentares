// Package ceap baixa e lê o arquivo anual da Cota para o Exercício da Atividade
// Parlamentar (CEAP) da Câmara, que contém as despesas de todos os deputados.
// É a segunda fonte externa do coleta-service (além da API REST de deputados).
package ceap

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
)

const (
	baseURLPadrao = "https://www.camara.leg.br/cotas"
	tamanhoLote   = 2000
)

// ErrIndisponivel marca falha ao baixar o arquivo do CEAP (rede, timeout, status).
var ErrIndisponivel = errors.New("arquivo do ceap indisponivel")

// Índices das colunas do CSV do CEAP (separador ';', 32 colunas).
const (
	colIdeCadastro = 2
	colDescricao   = 9
	colFornecedor  = 12
	colCNPJCPF     = 13
	colDataEmissao = 16
	colValorLiq    = 19
	colMes         = 20
	colAno         = 21
	colURLDoc      = 31
	totalColunas   = 32
)

type Client struct {
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

func NovoClient(baseURL string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = baseURLPadrao
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{
		baseURL:    baseURL,
		timeout:    timeout,
		httpClient: &http.Client{},
	}
}

// BaixarDespesas baixa o CSV do ano, faz streaming das linhas e entrega as
// despesas em lotes via aoReceberLote (evita carregar tudo em memória). Linhas
// sem ideCadastro (ex: lideranças) são ignoradas.
func (c *Client) BaixarDespesas(ctx context.Context, ano int, aoReceberLote func([]deputado.Despesa) error) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/Ano-%d.csv.zip", c.baseURL, ano)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("montar requisicao: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrIndisponivel, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d para o ano %d", ErrIndisponivel, resp.StatusCode, ano)
	}

	dados, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrIndisponivel, err)
	}

	return lerZip(dados, aoReceberLote)
}

func lerZip(dados []byte, aoReceberLote func([]deputado.Despesa) error) error {
	zr, err := zip.NewReader(bytes.NewReader(dados), int64(len(dados)))
	if err != nil {
		return fmt.Errorf("abrir zip do ceap: %w", err)
	}

	var csvFile *zip.File
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			csvFile = f
			break
		}
	}
	if csvFile == nil {
		return fmt.Errorf("csv nao encontrado no zip do ceap")
	}

	rc, err := csvFile.Open()
	if err != nil {
		return fmt.Errorf("abrir csv do ceap: %w", err)
	}
	defer rc.Close()

	r := csv.NewReader(rc)
	r.Comma = ';'
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	// descarta o cabeçalho
	if _, err := r.Read(); err != nil {
		return fmt.Errorf("ler cabecalho do ceap: %w", err)
	}

	lote := make([]deputado.Despesa, 0, tamanhoLote)
	for {
		registro, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ler linha do ceap: %w", err)
		}

		d, ok := linhaParaDespesa(registro)
		if !ok {
			continue
		}
		lote = append(lote, d)
		if len(lote) >= tamanhoLote {
			if err := aoReceberLote(lote); err != nil {
				return err
			}
			lote = make([]deputado.Despesa, 0, tamanhoLote)
		}
	}
	if len(lote) > 0 {
		if err := aoReceberLote(lote); err != nil {
			return err
		}
	}
	return nil
}

// linhaParaDespesa mapeia uma linha do CSV para Despesa. Retorna false quando a
// linha não tem um ideCadastro válido (não pertence a um deputado).
func linhaParaDespesa(campos []string) (deputado.Despesa, bool) {
	if len(campos) < totalColunas {
		return deputado.Despesa{}, false
	}

	id, err := strconv.ParseInt(strings.TrimSpace(campos[colIdeCadastro]), 10, 64)
	if err != nil || id <= 0 {
		return deputado.Despesa{}, false
	}

	return deputado.Despesa{
		DeputadoID:   id,
		Ano:          atoi(campos[colAno]),
		Mes:          atoi(campos[colMes]),
		Tipo:         strings.TrimSpace(campos[colDescricao]),
		Fornecedor:   strings.TrimSpace(campos[colFornecedor]),
		CNPJCPF:      strings.TrimSpace(campos[colCNPJCPF]),
		Valor:        parseValor(campos[colValorLiq]),
		Data:         soData(campos[colDataEmissao]),
		URLDocumento: strings.TrimSpace(campos[colURLDoc]),
	}, true
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func parseValor(s string) float64 {
	s = strings.TrimSpace(s)
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	// fallback para formato com vírgula decimal
	if v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64); err == nil {
		return v
	}
	return 0
}

func soData(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, 'T'); i > 0 {
		return s[:i]
	}
	return s
}
