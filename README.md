# E-Commerce API

API REST de e-commerce desenvolvida em **Go** com **Gin**, seguindo arquitetura limpa/hexagonal (`core` isolado de `adapters`), com autenticação via cookie `HttpOnly` (JWT), mensageria assíncrona com RabbitMQ, rate limiting distribuído em Redis, integração de pagamentos com Mercado Pago, chatbot via OpenRouter (Gemini 2.5 Flash Lite) e documentação interativa com Swagger.

---

## Principais Funcionalidades

- **Autenticação e Segurança**
  - Login/Logout com JWT (HS256) armazenado em cookie `HttpOnly` (`token`), com revogação de tokens.
  - Controle de acesso por perfil (`user` e `admin`).
  - Rate limiting por IP e por e-mail distribuído via Redis.
  - Proteção contra CSRF (`Origin`, `Referer` e `Sec-Fetch-Site`), headers de segurança e limite global de tamanho de payload (1 MB).
- **Usuários**
  - Cadastro assíncrono (RabbitMQ), verificação de e-mail por código, reenvio de código, recuperação/redefinição de senha e gestão de perfil.
- **Produtos**
  - CRUD completo (admin), listagem paginada e busca textual (`/api/v1/products/search`).
- **Carrinho**
  - Adicionar/remover itens, diminuir quantidade e selecionar itens individuais ou todos para checkout.
- **Pedidos e Estornos**
  - Checkout dos itens selecionados no carrinho, gestão de itens e endereço (CEP), histórico de status, cancelamento e fluxo de estorno (automático em `PREPARING`, bloqueado em `SHIPPED`, sujeito a aprovação admin em `DELIVERED`) com notificações por e-mail via RabbitMQ.
- **Pagamentos**
  - Pagamento via **Pix** e **Cartão de Crédito** (Mercado Pago ou gateway mock para testes), exigindo header `Idempotency-Key`.
  - Webhook do Mercado Pago com validação de assinatura **HMAC-SHA256** (`x-signature` / `x-request-id`) e sincronização assíncrona.
- **Chat (Assistente)**
  - Endpoint de chatbot integrado ao **OpenRouter** (`google/gemini-2.5-flash-lite`).

---

## Stack Tecnológica

- **Linguagem / Framework:** Go 1.27+ · [Gin](https://github.com/gin-gonic/gin)
- **Banco de Dados:** MySQL (`go-sql-driver/mysql`)
- **Cache / Rate Limit:** Redis (`go-redis/v9`)
- **Mensageria:** RabbitMQ (`amqp091-go`)
- **Pagamentos:** Mercado Pago (com opção de gateway mock local)
- **IA / Chatbot:** OpenRouter (`google/gemini-2.5-flash-lite`)
- **Documentação:** Swagger / OpenAPI 2.0 (`swaggo/swag`, `gin-swagger`, `swaggo/files`)

---

## Estrutura do Projeto

```text
e-commerce-api/
├── sql/
│   ├── script.sql                        # Schema inicial do banco MySQL
│   └── migrations/
│       └── 001_payment_refund_flow.sql   # Migration do fluxo de pagamentos e estornos
└── api/
    ├── cmd/
    │   └── main.go                       # Entrypoint, consumidores RabbitMQ e servidor HTTP
    ├── builder/                          # Injeção de dependências (controllers, use cases, consumers)
    ├── config/                           # Carregamento e validação de variáveis de ambiente (.env)
    ├── core/
    │   ├── apperrors/                    # Erros de domínio padronizados
    │   ├── entities/                     # Entidades e interfaces de repositório
    │   ├── ports/                        # Portas (e-mail, hash, mensageria, pagamentos, etc.)
    │   └── use_cases/                    # Regras de negócio por domínio
    ├── adapters/
    │   ├── controller/                   # Handlers HTTP, DTOs, mappers e helpers
    │   ├── database/                     # Conexão MySQL
    │   ├── email/                        # Envio de e-mails HTML (SMTP)
    │   ├── openrouter/                   # Cliente HTTP do OpenRouter (Gemini 2.5 Flash Lite)
    │   ├── payment/                      # Gateways de pagamento (Mercado Pago e Mock)
    │   ├── rabbitmq/                     # Produtores e consumidores RabbitMQ
    │   ├── redis/                        # Cliente Redis
    │   ├── repository/                   # Implementações MySQL dos repositórios
    │   ├── security/                     # Hash bcrypt e geração/validação de JWT
    │   └── server/                       # Middlewares e registro de rotas Gin
    ├── docs/                             # Documentação Swagger gerada (docs.go, swagger.json, swagger.yaml)
    ├── env_model                         # Modelo de variáveis de ambiente
    └── go.mod
```

---

## Pré-requisitos

- **Go** 1.27+
- **MySQL** (banco criado e scripts em `sql/` executados)
- **Redis**
- **RabbitMQ**
- **OpenRouter API Key** (opcional, necessária apenas para o endpoint `/api/v1/chat`)

---

## Configuração

1. Crie o banco de dados no MySQL e execute os scripts SQL na ordem:
   - `sql/script.sql`
   - `sql/migrations/001_payment_refund_flow.sql`

2. Na pasta `api/`, copie o arquivo `env_model` para `.env` e preencha os valores:

```powershell
cd e-commerce-api\api
Copy-Item env_model .env
```

### Variáveis de Ambiente

| Variável | Descrição |
| --- | --- |
| `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_HOST`, `DB_PORT` | Credenciais e endereço do MySQL |
| `API_PORT` | Porta HTTP da API (ex.: `:8080`) |
| `SECRET_KEY` | Chave secreta para assinatura JWT (**mínimo 32 caracteres**) |
| `REDIS_USER`, `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` | Configuração do Redis (padrão `localhost:6379`) |
| `EMAIL_HOST`, `EMAIL_PORT`, `EMAIL_USER`, `EMAIL_PASSWORD`, `EMAIL_FROM` | Configuração de e-mail (SMTP local ou remetente `EMAIL_FROM` na Brevo) |
| `BREVO_API_KEY` | Chave da API HTTP v3 da Brevo (`xkeysib-...`); quando preenchida, envia e-mails via HTTPS (porta 443) em vez de SMTP (recomendado no Render Free) |
| `RABBITMQ_USER`, `RABBITMQ_PASSWORD`, `RABBITMQ_HOST`, `RABBITMQ_PORT` | Configuração do RabbitMQ |
| `ACCESS_TOKEN_PAYMENT` | Access Token do Mercado Pago (deixe vazio para usar o gateway mock) |
| `MERCADO_PAGO_WEBHOOK_SECRET` | Segredo HMAC para validação de webhooks do Mercado Pago |
| `ALLOW_PAYMENT_SIMULATION` | `true` habilita a rota de simulação de pagamento quando `ACCESS_TOKEN_PAYMENT` está vazio |
| `KEY_OPEN_ROUTER`, `OPEN_ROUTER_MODEL` | Chave de API e modelo do OpenRouter (padrão `google/gemini-2.5-flash-lite`) |
| `COOKIE_SECURE` | `true` em produção HTTPS (usa `SameSite=None; Secure` para suportar front-end em outro domínio); `false` em desenvolvimento local HTTP (`SameSite=Lax`) |
| `COOKIE_DOMAIN` | Domínio do cookie de autenticação (vazio = host atual) |
| `CORS_ALLOWED_ORIGINS` | Origens permitidas separadas por vírgula (padrão `http://localhost:3000,https://nero-pi-teal.vercel.app`) |

---

## Como Executar

### Localmente

```powershell
cd e-commerce-api\api
go mod download
go run ./cmd/main.go
```

### Com Docker (API + MySQL + Redis + RabbitMQ)

1. Garanta que o arquivo `api/.env` esteja preenchido (a partir de `api/env_model`).
2. Na raiz de `e-commerce-api/`, suba todos os serviços passando `api/.env` para sincronizar as credenciais de banco, Redis e RabbitMQ:

```powershell
cd e-commerce-api
docker compose --env-file ./api/.env up -d --build
```

Para construir apenas a imagem da API:

```powershell
cd e-commerce-api
docker build -t e-commerce-api .
```

---

## Documentação da API (Swagger)

Com a API em execução, acesse a interface interativa do Swagger no navegador:

```text
http://localhost:<API_PORT>/swagger/index.html
```

Os arquivos OpenAPI gerados também podem ser importados diretamente no Swagger Editor, Postman ou Insomnia:

- `api/docs/swagger.json`
- `api/docs/swagger.yaml`

> **Dica para usar o botão "Try it out" em rotas protegidas:**
> 1. Execute `POST /api/v1/login` pelo próprio Swagger UI para que o navegador armazene o cookie `token`.
> 2. Inclua `http://localhost:<API_PORT>` em `CORS_ALLOWED_ORIGINS` no `.env` para evitar bloqueio `403 Forbidden` pelo `CSRFOriginMiddleware` em requisições `POST`/`PUT`/`DELETE`.

### Regenerando o Swagger

Sempre que alterar rotas, handlers ou DTOs, execute dentro de `e-commerce-api/api`:

```powershell
swag init -g main.go -d cmd,adapters/controller/helpers,adapters/controller/fieldErrors,adapters/controller/auth,adapters/controller/cart,adapters/controller/chat,adapters/controller/order,adapters/controller/payment,adapters/controller/product,adapters/controller/user -o docs
```

Caso ainda não tenha a CLI `swag` instalada:

```powershell
go install github.com/swaggo/swag/cmd/swag@latest
```

---

## Resumo dos Endpoints (`/api/v1`)

| Módulo | Rotas Principais |
| --- | --- |
| **Auth** | `POST /login`, `POST /logout` |
| **Users** | `POST /users`, `POST /users/verify`, `POST /users/new-verify-code`, `POST /users/reset-password`, `POST /users/confirm-reset-password`, `GET /users/me`, `PUT /users/:id`, `GET /users` (admin), `GET /users/:id` (admin), `DELETE /users/:id` (admin) |
| **Products** | `GET /products`, `GET /products/search`, `GET /products/:id`, `POST /products` (admin), `PUT /products/:id` (admin), `DELETE /products/:id` (admin) |
| **Cart** | `GET /cart`, `POST /cart`, `PUT /cart`, `PUT /cart/:id`, `PUT /cart/decrease/:id`, `DELETE /cart/:id` |
| **Orders** | `POST /orders`, `GET /orders/me`, `GET /orders/:id`, `PUT /orders/:id`, `PUT /orders/:id/cancel`, `POST /orders/:id/refund`, `GET /orders/:id/history`, `GET/POST /orders/:id/items`, `PUT/DELETE /orders/:id/items/:itemId`, `GET /orders` (admin), `PUT /orders/:id/status` (admin), `PUT /orders/:id/refund/approve` (admin), `PUT /orders/:id/refund/reject` (admin), `DELETE /orders/:id` (admin) |
| **Payments** | `POST /orders/:id/payments`, `GET /orders/:id/payment`, `POST /orders/:id/payment/confirm` (mock), `POST /payments/webhook` |
| **Chat** | `POST /chat` |

---

## Cartão de Teste (Mercado Pago)

Para testar pagamentos com cartão de crédito em ambiente sandbox do Mercado Pago:

- **Cartão (Mastercard):** `5480 8328 0103 3311`
- **Vencimento:** `11/30`
- **CVV:** `123`
- **Nome do titular:** `APRO`
- **CPF:** `12345678909`

---

## Testes e Verificação

```powershell
cd e-commerce-api\api
go test ./...
go vet ./...
go build ./...
```
