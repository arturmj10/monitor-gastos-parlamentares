// Package repository implementa a camada de acesso a dados do coleta-service
// usando GORM. É o único lugar que conhece detalhes de persistência.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormRepository implementa deputado.Repository com PostgreSQL/GORM.
type GormRepository struct {
	db *gorm.DB
}

func NovoGorm(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// Migrar cria/ajusta as tabelas a partir dos modelos (AutoMigrate).
func (r *GormRepository) Migrar() error {
	if err := r.db.AutoMigrate(&deputado.Deputado{}, &deputado.Despesa{}); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	return nil
}

// Salvar faz upsert dos deputados pelo id (insere ou atualiza).
func (r *GormRepository) Salvar(ctx context.Context, deputados []deputado.Deputado) error {
	if len(deputados) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).
		Create(&deputados).Error
	if err != nil {
		return fmt.Errorf("salvar deputados: %w", err)
	}
	return nil
}

// Listar retorna os deputados aplicando filtros opcionais de UF e partido.
func (r *GormRepository) Listar(ctx context.Context, filtro deputado.Filtro) ([]deputado.Deputado, error) {
	q := r.db.WithContext(ctx).Model(&deputado.Deputado{})
	if filtro.UF != "" {
		q = q.Where("uf = ?", filtro.UF)
	}
	if filtro.Partido != "" {
		q = q.Where("partido = ?", filtro.Partido)
	}

	var deputados []deputado.Deputado
	if err := q.Order("nome").Find(&deputados).Error; err != nil {
		return nil, fmt.Errorf("listar deputados: %w", err)
	}
	return deputados, nil
}

// Buscar retorna um deputado pelo id. Traduz gorm.ErrRecordNotFound para o erro
// de negócio deputado.ErrDeputadoNaoEncontrado, separando-o de erros de infra.
func (r *GormRepository) Buscar(ctx context.Context, id int64) (deputado.Deputado, error) {
	var d deputado.Deputado
	err := r.db.WithContext(ctx).First(&d, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return deputado.Deputado{}, deputado.ErrDeputadoNaoEncontrado
	}
	if err != nil {
		return deputado.Deputado{}, fmt.Errorf("buscar deputado: %w", err)
	}
	return d, nil
}

// SalvarDespesas substitui, numa transação, as despesas dos anos presentes no
// lote para aquele deputado (evita duplicar ao reconsultar a API).
func (r *GormRepository) SalvarDespesas(ctx context.Context, deputadoID int64, despesas []deputado.Despesa) error {
	if len(despesas) == 0 {
		return nil
	}
	anos := map[int]struct{}{}
	for _, d := range despesas {
		anos[d.Ano] = struct{}{}
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for ano := range anos {
			if err := tx.Where("deputado_id = ? AND ano = ?", deputadoID, ano).
				Delete(&deputado.Despesa{}).Error; err != nil {
				return fmt.Errorf("limpar despesas: %w", err)
			}
		}
		if err := tx.Create(&despesas).Error; err != nil {
			return fmt.Errorf("inserir despesas: %w", err)
		}
		return nil
	})
}

// LimparDespesasDoAno remove todas as despesas de um ano (antes de reimportar
// o arquivo do CEAP), tornando a importação idempotente.
func (r *GormRepository) LimparDespesasDoAno(ctx context.Context, ano int) error {
	if err := r.db.WithContext(ctx).Where("ano = ?", ano).Delete(&deputado.Despesa{}).Error; err != nil {
		return fmt.Errorf("limpar despesas do ano: %w", err)
	}
	return nil
}

// InserirLoteDespesas insere um lote de despesas em batches.
func (r *GormRepository) InserirLoteDespesas(ctx context.Context, despesas []deputado.Despesa) error {
	if len(despesas) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).CreateInBatches(despesas, 500).Error; err != nil {
		return fmt.Errorf("inserir lote de despesas: %w", err)
	}
	return nil
}

// ListarDespesas retorna as despesas de um deputado, filtrando por ano se ano > 0.
func (r *GormRepository) ListarDespesas(ctx context.Context, deputadoID int64, ano int) ([]deputado.Despesa, error) {
	q := r.db.WithContext(ctx).Where("deputado_id = ?", deputadoID)
	if ano > 0 {
		q = q.Where("ano = ?", ano)
	}

	var despesas []deputado.Despesa
	if err := q.Order("data DESC").Find(&despesas).Error; err != nil {
		return nil, fmt.Errorf("listar despesas: %w", err)
	}
	return despesas, nil
}
