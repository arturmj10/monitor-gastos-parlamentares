package deputado

import "errors"

// Erros de negócio do domínio. O handler os traduz em respostas 4xx,
// diferenciando-os de falhas de infraestrutura (que viram 5xx).
var (
	// ErrDeputadoNaoEncontrado: o deputado não existe (vira 404). O repositório
	// o retorna no lugar de gorm.ErrRecordNotFound.
	ErrDeputadoNaoEncontrado = errors.New("deputado nao encontrado")

	// ErrParametroInvalido: entrada inválida (ex: id <= 0) — vira 400.
	ErrParametroInvalido = errors.New("parametro invalido")
)
