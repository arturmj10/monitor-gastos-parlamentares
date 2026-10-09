// Package service contém a regra de negócio do coleta-service: orquestra a API
// externa (Câmara) e a persistência (repository), sem conhecer HTTP.
//
// As dependências são declaradas como interfaces (ClienteCamara e
// deputado.Repository), o que mantém o service desacoplado e testável com mocks.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
)

// ClienteCamara é a porta para a API externa da Câmara. É satisfeita por
// *camara.Client (injetado no main); nos testes, por um mock.
type ClienteCamara interface {
	ListarDeputados(ctx context.Context) ([]deputado.Deputado, error)
	BuscarDeputado(ctx context.Context, id int64) (deputado.Deputado, error)
	ListarDespesas(ctx context.Context, id int64, ano int) ([]deputado.Despesa, error)
}

// Service reúne as regras de negócio do coleta-service.
type Service struct {
	camara ClienteCamara
	repo   deputado.Repository
}

func Novo(camara ClienteCamara, repo deputado.Repository) *Service {
	return &Service{camara: camara, repo: repo}
}

// SincronizarDeputados busca a lista na API da Câmara e persiste no banco.
// Retorna quantos deputados foram sincronizados.
func (s *Service) SincronizarDeputados(ctx context.Context) (int, error) {
	deputados, err := s.camara.ListarDeputados(ctx)
	if err != nil {
		return 0, err // já classificado (negócio/infra) pelo client
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

// BuscarDeputado procura primeiro no banco. Só se não houver é que busca na API
// externa e faz cache local. Erro de infra do banco é propagado sem tentar a
// API, para não mascarar o problema.
func (s *Service) BuscarDeputado(ctx context.Context, id int64) (deputado.Deputado, error) {
	if id <= 0 {
		return deputado.Deputado{}, deputado.ErrParametroInvalido
	}

	d, err := s.repo.Buscar(ctx, id)
	if err == nil {
		return d, nil
	}
	if !errors.Is(err, deputado.ErrDeputadoNaoEncontrado) {
		return deputado.Deputado{}, err // erro de infraestrutura do banco
	}

	// não está no banco: busca na API externa e cacheia
	d, err = s.camara.BuscarDeputado(ctx, id)
	if err != nil {
		return deputado.Deputado{}, err
	}
	if err := s.repo.Salvar(ctx, []deputado.Deputado{d}); err != nil {
		return deputado.Deputado{}, fmt.Errorf("cachear deputado: %w", err)
	}
	return d, nil
}

// ObterDespesas busca as despesas de um deputado na API externa e as persiste.
func (s *Service) ObterDespesas(ctx context.Context, id int64, ano int) ([]deputado.Despesa, error) {
	if id <= 0 || ano < 0 {
		return nil, deputado.ErrParametroInvalido
	}

	despesas, err := s.camara.ListarDespesas(ctx, id, ano)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SalvarDespesas(ctx, id, despesas); err != nil {
		return nil, fmt.Errorf("salvar despesas: %w", err)
	}
	return despesas, nil
}
