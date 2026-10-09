package deputado

import "errors"

// ErrDeputadoNaoEncontrado é um erro de negócio: o deputado não existe na base.
// O repositório o retorna no lugar de gorm.ErrRecordNotFound, para que o
// handler o traduza em 404 e o diferencie de uma falha de infraestrutura
// (que vira 500).
var ErrDeputadoNaoEncontrado = errors.New("deputado nao encontrado")
