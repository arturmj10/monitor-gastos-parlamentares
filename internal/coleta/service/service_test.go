package service

import (
	"context"
	"errors"
	"testing"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- mocks (testify/mock) ---

type mockCamara struct{ mock.Mock }

func (m *mockCamara) ListarDeputados(ctx context.Context) ([]deputado.Deputado, error) {
	args := m.Called(ctx)
	var deps []deputado.Deputado
	if v := args.Get(0); v != nil {
		deps = v.([]deputado.Deputado)
	}
	return deps, args.Error(1)
}

func (m *mockCamara) BuscarDeputado(ctx context.Context, id int64) (deputado.Deputado, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(deputado.Deputado), args.Error(1)
}

func (m *mockCamara) ListarDespesas(ctx context.Context, id int64, ano int) ([]deputado.Despesa, error) {
	args := m.Called(ctx, id, ano)
	var desp []deputado.Despesa
	if v := args.Get(0); v != nil {
		desp = v.([]deputado.Despesa)
	}
	return desp, args.Error(1)
}

type mockRepo struct{ mock.Mock }

func (m *mockRepo) Salvar(ctx context.Context, deps []deputado.Deputado) error {
	return m.Called(ctx, deps).Error(0)
}

func (m *mockRepo) Listar(ctx context.Context, filtro deputado.Filtro) ([]deputado.Deputado, error) {
	args := m.Called(ctx, filtro)
	var deps []deputado.Deputado
	if v := args.Get(0); v != nil {
		deps = v.([]deputado.Deputado)
	}
	return deps, args.Error(1)
}

func (m *mockRepo) Buscar(ctx context.Context, id int64) (deputado.Deputado, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(deputado.Deputado), args.Error(1)
}

func (m *mockRepo) SalvarDespesas(ctx context.Context, id int64, desp []deputado.Despesa) error {
	return m.Called(ctx, id, desp).Error(0)
}

func (m *mockRepo) ListarDespesas(ctx context.Context, id int64, ano int) ([]deputado.Despesa, error) {
	args := m.Called(ctx, id, ano)
	var desp []deputado.Despesa
	if v := args.Get(0); v != nil {
		desp = v.([]deputado.Despesa)
	}
	return desp, args.Error(1)
}

// --- testes ---

func TestSincronizarDeputados(t *testing.T) {
	ctx := context.Background()
	deps := []deputado.Deputado{{ID: 1, Nome: "A"}, {ID: 2, Nome: "B"}}

	camara := new(mockCamara)
	repo := new(mockRepo)
	camara.On("ListarDeputados", ctx).Return(deps, nil)
	repo.On("Salvar", ctx, deps).Return(nil)

	svc := Novo(camara, repo)
	n, err := svc.SincronizarDeputados(ctx)

	require.NoError(t, err)
	assert.Equal(t, 2, n)
	camara.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestBuscarDeputado_VemDoBanco(t *testing.T) {
	ctx := context.Background()

	camara := new(mockCamara)
	repo := new(mockRepo)
	repo.On("Buscar", ctx, int64(7)).Return(deputado.Deputado{ID: 7, Nome: "Banco"}, nil)

	svc := Novo(camara, repo)
	d, err := svc.BuscarDeputado(ctx, 7)

	require.NoError(t, err)
	assert.Equal(t, "Banco", d.Nome)
	// não deve chamar a API externa quando já tem no banco
	camara.AssertNotCalled(t, "BuscarDeputado", mock.Anything, mock.Anything)
}

func TestBuscarDeputado_FallbackNaApiECacheia(t *testing.T) {
	ctx := context.Background()
	vindo := deputado.Deputado{ID: 7, Nome: "API"}

	camara := new(mockCamara)
	repo := new(mockRepo)
	repo.On("Buscar", ctx, int64(7)).Return(deputado.Deputado{}, deputado.ErrDeputadoNaoEncontrado)
	camara.On("BuscarDeputado", ctx, int64(7)).Return(vindo, nil)
	repo.On("Salvar", ctx, []deputado.Deputado{vindo}).Return(nil)

	svc := Novo(camara, repo)
	d, err := svc.BuscarDeputado(ctx, 7)

	require.NoError(t, err)
	assert.Equal(t, "API", d.Nome)
	camara.AssertExpectations(t)
	repo.AssertExpectations(t) // confirma que cacheou (Salvar foi chamado)
}

func TestBuscarDeputado_ErroDeInfraNaoTentaApi(t *testing.T) {
	ctx := context.Background()
	infra := errors.New("conexao recusada")

	camara := new(mockCamara)
	repo := new(mockRepo)
	repo.On("Buscar", ctx, int64(7)).Return(deputado.Deputado{}, infra)

	svc := Novo(camara, repo)
	_, err := svc.BuscarDeputado(ctx, 7)

	require.ErrorIs(t, err, infra)
	camara.AssertNotCalled(t, "BuscarDeputado", mock.Anything, mock.Anything)
}

func TestBuscarDeputado_IDInvalido(t *testing.T) {
	svc := Novo(new(mockCamara), new(mockRepo))
	_, err := svc.BuscarDeputado(context.Background(), 0)
	assert.ErrorIs(t, err, deputado.ErrParametroInvalido)
}

func TestObterDespesas_SalvaEDevolve(t *testing.T) {
	ctx := context.Background()
	desp := []deputado.Despesa{{DeputadoID: 7, Ano: 2024, Valor: 10}}

	camara := new(mockCamara)
	repo := new(mockRepo)
	camara.On("ListarDespesas", ctx, int64(7), 2024).Return(desp, nil)
	repo.On("SalvarDespesas", ctx, int64(7), desp).Return(nil)

	svc := Novo(camara, repo)
	out, err := svc.ObterDespesas(ctx, 7, 2024)

	require.NoError(t, err)
	assert.Len(t, out, 1)
	camara.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestObterDespesas_ParametroInvalido(t *testing.T) {
	svc := Novo(new(mockCamara), new(mockRepo))
	_, err := svc.ObterDespesas(context.Background(), 7, -1)
	assert.ErrorIs(t, err, deputado.ErrParametroInvalido)
}
