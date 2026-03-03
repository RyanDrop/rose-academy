# Rose Academy

Aplicação em **Go (Golang)** com arquitetura **MVC**, integrada ao **PostgreSQL** e **Abacate Pay** (criação de usuário, login e conta no gateway de pagamento).

## Estrutura do projeto

```
rose-academy/
├── api/              # RFC 9457 Problem Details (erros padronizados)
├── config/           # Configuração (DB, Abacate Pay, JWT)
├── controllers/      # Controllers (auth, goals)
├── db/               # Conexão PostgreSQL e ajustes de schema
├── middleware/       # JWT e auth
├── models/           # Entidades e repositórios (User, Goal, ABPCustomer)
├── routes/           # Rotas HTTP
├── services/         # Regras de negócio (Auth, cliente Abacate Pay)
├── go.mod
├── go.sum
└── main.go
```

## Como executar

1. Configure as variáveis de ambiente (veja tabela abaixo). **Obrigatório:** `ABACATE_PAY_API_KEY` para criar cliente na Abacate Pay no registro.

2. Instale dependências e suba o servidor:

   ```bash
   go mod tidy
   go run main.go
   ```

3. API disponível em `http://localhost:8080`.
4. **Swagger UI:** abra `http://localhost:8080/swagger/index.html` no navegador para ver e testar a documentação da API.

## API – Primeira parte (auth + Abacate Pay)

| Método | Rota             | Descrição                           | Auth   |
| ------ | ---------------- | ----------------------------------- | ------ |
| GET    | `/health`        | Health check                        | Não    |
| GET    | `/goals`         | Lista metas (para formulário)       | Não    |
| POST   | `/auth/register` | Criar usuário + cliente Abacate Pay | Não    |
| POST   | `/auth/login`    | Login (retorna usuário + token)     | Não    |
| GET    | `/auth/me`       | Dados do usuário logado             | Bearer |

### POST /auth/register

Body (JSON):

- **Obrigatórios:** `email`, `password`, `full_name`, `cellphone`, `tax_id` (CPF/CNPJ).
- **Opcionais:** `goal_id` (UUID de `goals`), `gender` (`feminino` | `masculino` | `outro` | `prefiro_nao_dizer`).

Resposta (201): `{ "user": { ... }, "token": "jwt..." }`.

Fluxo: insere em `users`, chama Abacate Pay `POST /customer/create`, insere em `abp_customers`.

### POST /auth/login

Body (JSON): `email`, `password`.

Resposta (200): `{ "user": { ... }, "token": "jwt..." }`. Atualiza `last_login_at`.

### GET /auth/me

Header: `Authorization: Bearer <token>`.

Resposta (200): objeto do usuário (sem `password_hash`).

## Erros (RFC 9457)

As respostas de erro seguem **Problem Details for HTTP APIs** (RFC 9457): `Content-Type: application/problem+json`, com `type`, `title`, `status`, `detail` e `instance` (path da requisição). Para validação com múltiplos campos, é retornado também o array `errors` com `field` e `message`.

Exemplo (400):

```json
{
  "type": "about:blank",
  "title": "Bad Request",
  "status": 400,
  "detail": "body JSON inválido",
  "instance": "/auth/register"
}
```

Exemplo (422 – validação):

```json
{
  "type": "about:blank",
  "title": "Validation Failed",
  "status": 422,
  "detail": "Um ou mais campos possuem valores inválidos.",
  "instance": "/auth/register",
  "errors": [
    { "field": "email", "message": "é obrigatório" },
    { "field": "tax_id", "message": "é obrigatório" }
  ]
}
```

Erros 500 (incluindo panic) não expõem detalhes internos; o `detail` é genérico.

## Variáveis de ambiente

| Variável                     | Descrição                                             | Padrão                                      |
| ---------------------------- | ----------------------------------------------------- | ------------------------------------------- |
| `PORT`                       | Porta do servidor                                     | `8080`                                      |
| `ENV`                        | Ambiente                                              | `development`                               |
| `DB_HOST`                    | Host PostgreSQL                                       | `localhost`                                 |
| `DB_PORT`                    | Porta PostgreSQL                                      | `5432`                                      |
| `DB_USER`                    | Usuário do banco                                      | `admin`                                     |
| `DB_PASSWORD`                | Senha do banco                                        | `password`                                  |
| `DB_NAME`                    | Nome do banco                                         | `rose_gym_db`                               |
| `DB_SSLMODE`                 | SSL                                                   | `disable`                                   |
| `ABACATE_PAY_API_KEY`        | Chave API Abacate Pay                                 | (obrigatório para registro)                 |
| `ABACATE_PAY_BASE_URL`       | Base URL da API                                       | `https://api.abacatepay.com/v1`             |
| `ABACATE_PAY_WEBHOOK_SECRET` | Secret do webhook (igual ao configurado no dashboard) | (obrigatório para atualizar banco ao pagar) |
| `JWT_SECRET`                 | Segredo para assinatura JWT                           | `change-me-in-production`                   |
| `JWT_EXPIRATION_HOURS`       | Expiração do token (horas)                            | `72`                                        |

## Stack

- **Router:** [Chi](https://github.com/go-chi/chi)
- **Docs:** [Swaggo Swagger](https://github.com/swaggo/swag) — gere de novo com `go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g main.go --parseDependency --parseInternal` após alterar anotações.
- **Banco:** PostgreSQL com [pgx](https://github.com/jackc/pgx)
- **Auth:** JWT ([golang-jwt/jwt](https://github.com/golang-jwt/jwt)), bcrypt
- **Pagamentos:** [Abacate Pay](https://docs.abacatepay.com/) (criação de cliente no registro)

## Arquitetura MVC

- **Model:** `models/` — entidades (User, Goal, ABPCustomer) e repositórios (persistência).
- **Service:** `services/` — regras de negócio (registro, login, chamada à API Abacate Pay).
- **Controller:** `controllers/` — HTTP (body, status, JSON); chama services e models.
- **Routes:** `routes/routes.go` — monta router e injeta config, DB e middlewares.

O banco segue o DDL do projeto (incl. `users`, `goals`, `abp_customers`). As colunas `cellphone` e `tax_id` são adicionadas em `users` na primeira conexão, se não existirem, para atender à API Abacate Pay.
