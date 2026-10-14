# Monitor de Gastos Parlamentares

Sistema de microsserviços em Go que coleta dados abertos da Câmara dos Deputados.
Este repositório contém o `coleta-service`: consome duas fontes externas — a API
REST de deputados e o arquivo anual de despesas (CEAP) — persiste em PostgreSQL
próprio e expõe uma API REST.

## Requisitos

- Go 1.25+
- Docker e Docker Compose

## Como rodar

Com Docker Compose (serviço + banco):

```bash
docker compose up --build
```

A API sobe em `http://localhost:8080`.

Para rodar só o banco e o serviço localmente:

```bash
docker compose up -d postgres
```

```bash
DATABASE_URL="postgres://coleta:coleta@localhost:5433/coleta?sslmode=disable" go run ./cmd/coleta
```

## Fluxo de uso

1. Importar os deputados da API da Câmara:
   ```bash
   curl -X POST http://localhost:8080/deputados/sync
   ```
2. Importar as despesas de um ano (baixa o arquivo CEAP e ingere no banco):
   ```bash
   curl -X POST "http://localhost:8080/despesas/sync?ano=2024"
   ```
3. Consultar:
   ```bash
   curl "http://localhost:8080/deputados?uf=SP"
   curl http://localhost:8080/deputados/204379
   curl "http://localhost:8080/deputados/204379/despesas?ano=2024"
   ```

As despesas são lidas do banco; o download do CEAP só ocorre no `POST /despesas/sync`.

## Variáveis de ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `PORT` | `8080` | Porta HTTP do serviço |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error` |
| `DATABASE_URL` | — (obrigatória) | DSN do PostgreSQL |
| `CAMARA_BASE_URL` | API da Câmara v2 | Base da API REST de deputados |
| `CAMARA_TIMEOUT` | `10s` | Timeout por requisição à Câmara |
| `CEAP_BASE_URL` | portal de cotas da Câmara | Base dos arquivos de despesas (CEAP) |
| `CEAP_TIMEOUT` | `120s` | Timeout do download do arquivo CEAP |

## Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/healthz` | Health check |
| `POST` | `/deputados/sync` | Importa os deputados da API da Câmara |
| `GET` | `/deputados` | Lista os deputados (filtros `?uf=` e `?partido=`) |
| `GET` | `/deputados/{id}` | Busca um deputado (busca na API se não estiver no banco) |
| `POST` | `/despesas/sync?ano=` | Importa as despesas do ano a partir do arquivo CEAP |
| `GET` | `/deputados/{id}/despesas?ano=` | Despesas do deputado (lidas do banco) |

## Testes

```bash
go test ./...
```

## Estrutura

```
cmd/coleta                  entrada do serviço
internal/coleta/config      configuração por variáveis de ambiente
internal/coleta/httpapi     camada HTTP (chi): rotas, handlers, DTOs
internal/coleta/service     regras de negócio
internal/coleta/deputado    domínio (modelos, erros, interface do repositório)
internal/coleta/repository  acesso a dados (GORM/Postgres)
internal/coleta/storage     conexão com o banco
internal/coleta/camara      client REST da API de deputados
internal/coleta/ceap        download e leitura do arquivo de despesas (CEAP)
```
