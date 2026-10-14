// Package service contém a regra de negócio do coleta-service: orquestra as
// fontes externas (API da Câmara e arquivo do CEAP) e a persistência, sem
// conhecer HTTP. As dependências são interfaces, testáveis com mocks.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
)

// ClienteCamara é a porta para a API REST de deputados da Câmara.
type ClienteCamara interface {
	ListarDeputados(ctx context.Context) ([]deputado.Deputado, error)
	BuscarDeputado(ctx context.Context, id int64) (deputado.Deputado, error)
}

// FonteDespesas é a porta para o arquivo anual de despesas (CEAP). Entrega as
// despesas em lotes via callback.
type FonteDespesas interface {
	BaixarDespesas(ctx context.Context, ano int, aoReceberLote func([]deputado.Despesa) error) error
}

type Service struct {
	camara ClienteCamara
	ceap   FonteDespesas
	repo   deputado.Repository
}

func Novo(camara ClienteCamara, ceap FonteDespesas, repo deputado.Repository) *Service {
	return &Service{camara: camara, ceap: ceap, repo: repo}
}

// SincronizarDeputados busca a lista na API da Câmara e persiste no banco.
func (s *Service) SincronizarDeputados(ctx context.Context) (int, error) {
	deputados, err := s.camara.ListarDeputados(ctx)
	if err != nil {
		return 0, err
	}
	if err := s.repo.Salvar(ctx, deputados); err != nil {
		return 0, fmt.Errorf("salvar deputados: %w", err)
	}
	return len(deputados), nil
}

// ListarDeputados devolve os deputados já persistidos, aplicando filtros.
func (s *Service) ListarDeputados(ctx context.Context, filtro deputado.Filtro) ([]deputado.Deputado, error) {
	return s.repo.Listar(ctx, filtro)
}

// BuscarDeputado procura no banco; se não houver, busca na API e cacheia.
func (s *Service) BuscarDeputado(ctx context.Context, id int64) (deputado.Deputado, error) {
	if id <= 0 {
		return deputado.Deputado{}, deputado.ErrParametroInvalido
	}

	d, err := s.repo.Buscar(ctx, id)
	if err == nil {
		return d, nil
	}
	if !errors.Is(err, deputado.ErrDeputadoNaoEncontrado) {
		return deputado.Deputado{}, err
	}

	d, err = s.camara.BuscarDeputado(ctx, id)
	if err != nil {
		return deputado.Deputado{}, err
	}
	if err := s.repo.Salvar(ctx, []deputado.Deputado{d}); err != nil {
		return deputado.Deputado{}, fmt.Errorf("cachear deputado: %w", err)
	}
	return d, nil
}

// SincronizarDespesas baixa o arquivo do CEAP do ano e o ingere no banco (limpa
// o ano antes, para ser idempotente). Retorna quantas despesas foram importadas.
func (s *Service) SincronizarDespesas(ctx context.Context, ano int) (int, error) {
	if ano <= 0 {
		return 0, deputado.ErrParametroInvalido
	}
	if err := s.repo.LimparDespesasDoAno(ctx, ano); err != nil {
		return 0, fmt.Errorf("limpar despesas do ano: %w", err)
	}

	total := 0
	err := s.ceap.BaixarDespesas(ctx, ano, func(lote []deputado.Despesa) error {
		if err := s.repo.InserirLoteDespesas(ctx, lote); err != nil {
			return err
		}
		total += len(lote)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

// ObterDespesas lê as despesas de um deputado já persistidas (ingeridas do CEAP).
func (s *Service) ObterDespesas(ctx context.Context, id int64, ano int) ([]deputado.Despesa, error) {
	if id <= 0 || ano < 0 {
		return nil, deputado.ErrParametroInvalido
	}
	return s.repo.ListarDespesas(ctx, id, ano)
}
