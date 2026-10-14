package deputado

import "context"

// Repository é a porta de acesso a dados do domínio. O service depende desta
// interface (nunca de uma struct concreta), o que permite trocar a
// implementação (GORM, memória, mock nos testes).
type Repository interface {
	Salvar(ctx context.Context, deputados []Deputado) error
	Listar(ctx context.Context, filtro Filtro) ([]Deputado, error)
	Buscar(ctx context.Context, id int64) (Deputado, error)
	ListarDespesas(ctx context.Context, deputadoID int64, ano int) ([]Despesa, error)

	// carga em lote das despesas do CEAP (arquivo anual)
	LimparDespesasDoAno(ctx context.Context, ano int) error
	InserirLoteDespesas(ctx context.Context, despesas []Despesa) error
}
