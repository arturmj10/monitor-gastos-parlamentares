package deputado

import "time"

// Deputado é o modelo de domínio de um parlamentar. As tags gorm controlam o
// mapeamento para a tabela; o id é o mesmo da API da Câmara (não é gerado pelo
// banco).
type Deputado struct {
	ID           int64     `gorm:"primaryKey;autoIncrement:false"`
	Nome         string    `gorm:"not null"`
	Partido      string    `gorm:"index"`
	UF           string    `gorm:"index"`
	Email        string    `gorm:"default:''"`
	URLFoto      string    `gorm:"default:''"`
	CriadoEm     time.Time `gorm:"autoCreateTime"`
	AtualizadoEm time.Time `gorm:"autoUpdateTime"`
}

func (Deputado) TableName() string { return "deputados" }

// Despesa é um gasto da cota parlamentar de um deputado. O id é gerado pelo
// banco (não vem da Câmara).
type Despesa struct {
	ID           uint      `gorm:"primaryKey"`
	DeputadoID   int64     `gorm:"index;not null"`
	Ano          int       `gorm:"index"`
	Mes          int       `gorm:"default:0"`
	Tipo         string    `gorm:"default:''"`
	Fornecedor   string    `gorm:"default:''"`
	CNPJCPF      string    `gorm:"default:''"`
	Valor        float64   `gorm:"default:0"`
	Data         string    `gorm:"default:''"`
	URLDocumento string    `gorm:"default:''"`
	CriadoEm     time.Time `gorm:"autoCreateTime"`
}

func (Despesa) TableName() string { return "despesas" }

// Filtro são os filtros opcionais da listagem de deputados.
type Filtro struct {
	UF      string
	Partido string
}
