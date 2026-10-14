// Package camara é o client REST da API externa de Dados Abertos da Câmara
// dos Deputados. É aqui que vive a "comunicação síncrona externa": requisições
// com context+timeout, fechamento do Body e verificação do StatusCode.
package camara

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
)

const baseURLPadrao = "https://dadosabertos.camara.leg.br/api/v2"

// ErrIndisponivel marca falha de infraestrutura ao falar com a API da Câmara
// (rede, timeout, status 5xx). O handler o traduz em 502 — distinto de um
// erro de negócio como deputado.ErrDeputadoNaoEncontrado (404).
var ErrIndisponivel = errors.New("api da camara indisponivel")

// Client consome a API da Câmara. baseURL e timeout são injetáveis, o que
// permite apontar para um servidor de teste (httptest) sem mudar o código.
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
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL:    baseURL,
		timeout:    timeout,
		httpClient: &http.Client{},
	}
}

// --- formato da resposta da API da Câmara (DTOs internos) ---

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type respDeputados struct {
	Dados []struct {
		ID           int64  `json:"id"`
		Nome         string `json:"nome"`
		SiglaPartido string `json:"siglaPartido"`
		SiglaUf      string `json:"siglaUf"`
		URLFoto      string `json:"urlFoto"`
		Email        string `json:"email"`
	} `json:"dados"`
	Links []link `json:"links"`
}

type respDeputado struct {
	Dados struct {
		ID           int64 `json:"id"`
		UltimoStatus struct {
			Nome         string `json:"nome"`
			SiglaPartido string `json:"siglaPartido"`
			SiglaUf      string `json:"siglaUf"`
			URLFoto      string `json:"urlFoto"`
			Email        string `json:"email"`
		} `json:"ultimoStatus"`
	} `json:"dados"`
}

// ListarDeputados busca todos os deputados em exercício, seguindo a paginação.
func (c *Client) ListarDeputados(ctx context.Context) ([]deputado.Deputado, error) {
	q := url.Values{
		"ordem":      {"ASC"},
		"ordenarPor": {"nome"},
		"itens":      {"100"},
	}
	proxima := c.baseURL + "/deputados?" + q.Encode()

	var deputados []deputado.Deputado
	for proxima != "" {
		var resp respDeputados
		if err := c.get(ctx, proxima, &resp); err != nil {
			return nil, err
		}
		for _, d := range resp.Dados {
			deputados = append(deputados, deputado.Deputado{
				ID:      d.ID,
				Nome:    d.Nome,
				Partido: d.SiglaPartido,
				UF:      d.SiglaUf,
				Email:   d.Email,
				URLFoto: d.URLFoto,
			})
		}
		proxima = hrefPorRel(resp.Links, "next")
	}
	return deputados, nil
}

// BuscarDeputado busca um deputado pelo id. Retorna
// deputado.ErrDeputadoNaoEncontrado quando a API responde 404.
func (c *Client) BuscarDeputado(ctx context.Context, id int64) (deputado.Deputado, error) {
	endpoint := c.baseURL + "/deputados/" + strconv.FormatInt(id, 10)

	var resp respDeputado
	if err := c.get(ctx, endpoint, &resp); err != nil {
		return deputado.Deputado{}, err
	}

	us := resp.Dados.UltimoStatus
	return deputado.Deputado{
		ID:      resp.Dados.ID,
		Nome:    us.Nome,
		Partido: us.SiglaPartido,
		UF:      us.SiglaUf,
		Email:   us.Email,
		URLFoto: us.URLFoto,
	}, nil
}

// get faz a requisição GET: aplica timeout via context, define o Accept,
// SEMPRE fecha o Body e confere o StatusCode antes de decodificar.
func (c *Client) get(ctx context.Context, endpoint string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("montar requisicao: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// cobre falha de rede e timeout (context deadline)
		return fmt.Errorf("%w: %v", ErrIndisponivel, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return deputado.ErrDeputadoNaoEncontrado
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("%w: status %d", ErrIndisponivel, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decodificar resposta: %w", err)
	}
	return nil
}

func hrefPorRel(links []link, rel string) string {
	for _, l := range links {
		if l.Rel == rel {
			return l.Href
		}
	}
	return ""
}
