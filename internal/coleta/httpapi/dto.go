package httpapi

import "github.com/arturmj10/monitor-gastos-parlamentares/internal/coleta/deputado"

type deputadoResp struct {
	ID      int64  `json:"id"`
	Nome    string `json:"nome"`
	Partido string `json:"partido"`
	UF      string `json:"uf"`
	Email   string `json:"email,omitempty"`
	URLFoto string `json:"urlFoto,omitempty"`
}

type despesaResp struct {
	Ano          int     `json:"ano"`
	Mes          int     `json:"mes"`
	Tipo         string  `json:"tipo"`
	Fornecedor   string  `json:"fornecedor"`
	CNPJCPF      string  `json:"cnpjCpf"`
	Valor        float64 `json:"valor"`
	Data         string  `json:"data"`
	URLDocumento string  `json:"urlDocumento,omitempty"`
}

func toDeputadoResp(d deputado.Deputado) deputadoResp {
	return deputadoResp{
		ID:      d.ID,
		Nome:    d.Nome,
		Partido: d.Partido,
		UF:      d.UF,
		Email:   d.Email,
		URLFoto: d.URLFoto,
	}
}

func toDeputadosResp(ds []deputado.Deputado) []deputadoResp {
	out := make([]deputadoResp, 0, len(ds))
	for _, d := range ds {
		out = append(out, toDeputadoResp(d))
	}
	return out
}

func toDespesasResp(ds []deputado.Despesa) []despesaResp {
	out := make([]despesaResp, 0, len(ds))
	for _, d := range ds {
		out = append(out, despesaResp{
			Ano:          d.Ano,
			Mes:          d.Mes,
			Tipo:         d.Tipo,
			Fornecedor:   d.Fornecedor,
			CNPJCPF:      d.CNPJCPF,
			Valor:        d.Valor,
			Data:         d.Data,
			URLDocumento: d.URLDocumento,
		})
	}
	return out
}
