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
| PATCH  | `/auth/me`       | Atualizar perfil (campos opcionais) | Bearer |

### POST /auth/register

Body (JSON):

- **Obrigatórios:** `email`, `password`, `full_name`. **Opcional:** `tax_id` (CPF/CNPJ; só dígitos são gravados).

Resposta (201): `{ "user": { ... }, "token": "jwt..." }`.

Fluxo: insere em `users`, chama Abacate Pay **`POST /v2/customers/create`** com JSON plano (`email` obrigatório; também `name` quando houver; celular e CPF podem ir depois no perfil), insere em `abp_customers`. A chave do painel é da API v2 — `/v1/customer/create` gerava *API key version mismatch*.

CPF também pode ir no **registro** (`tax_id` opcional). Celular, CPF (atualizar), meta (`goal_id`) e gênero em **PATCH /auth/me** (autenticado). `tax_id` gravado só com dígitos; `gender`: `feminino` | `masculino` | `outro` | `prefiro_nao_dizer` (string vazia remove o campo).

### POST /auth/login

Body (JSON): `email`, `password`.

Resposta (200): `{ "user": { ... }, "token": "jwt..." }`. Atualiza `last_login_at`.

### GET /auth/me

Header: `Authorization: Bearer <token>`.

Resposta (200): objeto do usuário (sem `password_hash`).

### PATCH /auth/me

Header: `Authorization: Bearer <token>`. Body (JSON): todos opcionais — entre outros, `cellphone`, `tax_id`, `goal_id`, `gender`, `date_birth`, `weight`, `height`, `profile_img`, `email_verified`. Resposta (200): usuário atualizado. **409** se `tax_id` já estiver em uso por outra conta.

## Assinaturas Abacate Pay (API v2)

A Abacate Pay expõe [checkout de assinatura recorrente](https://docs.abacatepay.com/pages/subscriptions/reference) (`POST /v2/subscriptions/create`). O produto precisa ter **ciclo** (MONTHLY, YEARLY, etc.) cadastrado na loja Abacate Pay.

Na tabela `products`, preencha **`abacate_subscription_product_id`** com o id do produto na Abacate Pay (ex.: `prod_xxx`), ou envie **`abp_product_id`** no body ao criar o checkout.

| Método | Rota | Descrição | Auth |
| ------ | ---- | --------- | ---- |
| POST | `/abacate/subscriptions/checkout` | Cria checkout de assinatura (v2); body: `product_id`, `return_url`, `completion_url`, opcional `abp_product_id`, `external_id`, `methods` | Bearer |
| GET | `/abacate/subscriptions` | Lista `user_subscriptions` do usuário (sem API Abacate); no máximo uma linha **PAID** na resposta | Bearer |
| POST | `/abacate/subscriptions/cancel` | Cancela assinatura; body: `{ "id": "subs_..." }` | Bearer |
| GET | `/subscriptions/active` | Gate: `{ "status": "PAID" \| "PENDING" \| null }` só com `user_subscriptions` do usuário (PAID > PENDING) | Bearer |

Variáveis: `ABACATE_PAY_V2_BASE_URL` (padrão `https://api.abacatepay.com/v2`), `ABACATE_PAY_SUBSCRIPTION_METHODS` (padrão `CARD`; pode ser `PIX` ou `PIX,CARD` conforme a loja).

O fluxo **POST /subscribe** continua sendo cobrança **avulsa** (v1 `billing/create`). Use **checkout de assinatura** quando precisar de recorrência gerenciada pela Abacate Pay.

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
    { "field": "full_name", "message": "é obrigatório" }
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
| `ABACATE_PAY_BASE_URL`       | Base URL da API (v1 cobrança avulsa)                  | `https://api.abacatepay.com/v1`             |
| `ABACATE_PAY_V2_BASE_URL`    | Base URL API v2 (assinaturas)                         | `https://api.abacatepay.com/v2`             |
| `ABACATE_PAY_METHODS`        | PIX e/ou CARD na cobrança `/subscribe`                | `PIX`                                       |
| `ABACATE_PAY_SUBSCRIPTION_METHODS` | Métodos no checkout v2 `/abacate/subscriptions/checkout` | `CARD`                                |
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
