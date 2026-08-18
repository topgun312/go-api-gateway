# API Gateway с Rate Limiter

Прокси-сервер (API Gateway) на Go: принимает HTTP-запросы, проверяет API-ключ в
PostgreSQL, ограничивает частоту запросов (не более 5 в минуту на ключ) через
Redis и проксирует валидные запросы в локальный ИИ-сервис **Ollama**.

```
+--------+  HTTP/1.1 +-----------------------+  HTTP/1.1 +--------+
| Client | ─────────▶ |   API Gateway (Go)   | ─────────▶ | Ollama |
| (curl) |            |  ┌─────────────┐     |            +--------+
|        |            |  │ Rate Limiter │     |
|        |            |  └──────┬──────┘     |
|        |            │         │ Redis      │   ← счётчики лимита
|        |            │   ┌─────▼───────┐    |
|        |            │   │   Redis     │    |
|        |            │   └─────────────┘    |
|        |            │         │ SQL        │
|        |            │   ┌─────▼───────┐    │
|        |            │   │ PostgreSQL  │    │   ← проверка API-ключа
|        |            │   └─────────────┘    │
+--------+            +----------------------+
```

## Стек

| Компонент | Технология | Зачем |
|---|---|---|
| Язык | Go 1.25 (`net/http` stdlib) | HTTP-сервер, проксирование |
| Хранилище ключей | PostgreSQL 16 | Таблица `users` — валидные API-ключи |
| Rate limit | Redis 7 | Счётчики запросов: `INCR` + `EXPIRE` (fixed window) |
| ИИ-сервис | Ollama | `POST /api/generate`, `POST /api/chat` |
| Запуск | Docker Compose | Postgres, Redis, Ollama и сам шлюз |

## Как работает

Обработчик шлюза выполняет цепочку `auth → ratelimit → proxy`:

1. **Auth** — извлекает ключ из `Authorization: Bearer <key>` или заголовка
   `X-API-Key` и ищет его в PostgreSQL. Нет/неверный ключ → `401`.
2. **Rate limit** — атомарный Lua-скрипт `INCR` + `EXPIRE` (TTL 60 сек) в Redis.
   Ключ `ratelimit:<api_key>:<минута>` — фиксированное окно в 1 минуту.
   Лимит исчерпан → `429 Too Many Requests` + заголовки `X-RateLimit-*`,
   `Retry-After: 60`.
3. **Proxy** — валидный запрос пересылается в Ollama (`/api/generate`),
   ответ и статус пробрасываются клиенту как есть.

## Структура проекта

```
go-api-gateway/
├── cmd/
│   └── gateaway/
│       └── main.go              # точка входа, сборка компонентов
├── internal/
│   ├── config/config.go         # конфиг из env-переменных
│   ├── models/models.go         # структура User
│   ├── storage/
│   │   ├── postgres.go          # подключение к БД + GetUserByAPIKey
│   │   └── redis.go             # клиент Redis
│   ├── ratelimit/limiter.go     # Limiter + RedisLimiter (Lua INCR/EXPIRE)
│   ├── proxy/proxy.go           # проксирование в Ollama
│   └── handlers/gateway.go      # HTTP-обработчик: auth → limit → proxy
├── migrations/schema.sql        # схема БД (выполняется при первом старте)
├── Dockerfile                   # сборка шлюза
├── docker-compose.yml           # postgres + redis + ollama + app
└── go.mod / go.sum
```

## Требования

- [Docker](https://docs.docker.com/get-docker/) + Docker Compose v2
- Go 1.25 (только если запускаешь без Docker)

## Быстрый старт

```bash
git clone <repo> && cd go-api-gateway

# 1. Поднять всё (postgres, redis, шлюз)
docker compose up -d --build

# 2. Выкачать модель в Ollama (первый раз, если Ollama в Docker)
docker compose exec ollama ollama pull llama3.2
```

Шлюз слушает `http://localhost:8080`. Миграция `migrations/schema.sql`
выполняется Postgres автоматически при **первом** старте и создаёт тестовый
ключ `test-key-123`.

> **Про сеть шлюза.** Сервис `app` в `docker-compose.yml` использует
> `network_mode: host`, поэтому ходит в Postgres/Redis по `localhost:5432` /
> `localhost:6379`, а в Ollama — по `localhost:11434`. Это удобно, когда Ollama
> установлен на хосте нативно (как в этой среде). Если Ollama должен жить в
> Docker — убери `network_mode: host`, верни проброс порта и укажи
> `OLLAMA_URL: http://ollama:11434`.

> Если БД уже была инициализирована без миграций, примени схему вручную:
> ```bash
> docker compose exec postgres psql -U gateway -d gateway \
>   -f /docker-entrypoint-initdb.d/schema.sql
> ```

## Запуск без Docker (для разработки)

```bash
go mod tidy
go run ./cmd/gateaway
```

Требуются запущенные Postgres (`:5432`), Redis (`:6379`) и Ollama (`:11434`).

## Конфигурация

Все настройки — через env-переменные (значения по умолчанию в скобках):

| Переменная | Описание | По умолчанию |
|---|---|---|
| `SERVER_PORT` | Адрес HTTP-сервера | `:8080` |
| `DB_DSN` | DSN PostgreSQL | `postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable` |
| `REDIS_ADDR` | Адрес Redis | `localhost:6379` |
| `OLLAMA_URL` | Базовый URL Ollama | `http://localhost:11434` |
| `MAX_REQUESTS` | Лимит запросов в минуту | `5` |

> Внутри docker-compose адреса автоматически указывают на сервисы
> (`postgres`, `redis`, `ollama`) — см. `environment` сервиса `app`.

### Переменные через `.env`

Для локальной разработки значения можно вынести в файл `.env` (уже в
`.gitignore`):

```bash
cp .env.example .env   # затем отредактируй под себя
```

Docker Compose автоматически подхватит `.env` для `${VAR}`-подстановки, а
`config.go` при локальном `go run` загружает его через `godotenv`. Секреты
(пароль Postgres, DSN) в git не попадут.

## Примеры запросов

Валидный запрос:

```bash
curl http://localhost:8080/api/generate \
  -H "Authorization: Bearer test-key-123" \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3.2","prompt":"Привет!","stream":false}'
```

Неверный ключ → `401`:

```bash
curl -i http://localhost:8080/api/generate -H "Authorization: Bearer bad-key" \
  -H "Content-Type: application/json" -d '{"model":"llama3.2","prompt":"x","stream":false}'
```

> Лимит — 5 запросов в минуту на ключ. При превышении → `429` с заголовками
> `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After: 60`.

Проверка лимита — 7 запросов подряд (6-й и 7-й должны вернуть `429`):

```bash
for i in $(seq 1 7); do
  code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/generate \
    -H "Authorization: Bearer test-key-123" \
    -H "Content-Type: application/json" \
    -d '{"model":"llama3.2","prompt":"test","stream":false}')
  echo "request $i -> $code"
done
# ожидаемый вывод: request 1-5 -> 200, request 6-7 -> 429
```

> Учти: Ollama отвечает ~6 секунд на запрос, поэтому в curl-цикле 6 запросов
> могут растянуться через границу минуты, и 429 не выпадет (fixed window).
> Чтобы гарантированно увидеть `429`, гони запросы в один календарный
> промежуток или используй `go test` (см. раздел «Тестирование»).

## Тестирование

Проект покрыт **модульными** (unit) и **интеграционными** (integration) тестами.
Unit-тесты не требуют внешних зависимостей: Redis заменяется in-memory `miniredis`,
апстрим — `httptest`, БД и лимитер — моками. Интеграционные требуют живого PostgreSQL.

### Какие тесты реализованы

| Файл | Тип | Тест-кейсы |
|---|---|---|
| `internal/config/config_test.go` | unit | дефолтные значения; переопределение через env; невалидный `MAX_REQUESTS` → дефолт |
| `internal/handlers/gateway_test.go` | unit | 401 без ключа; 401 неверный ключ; 500 ошибка БД; 429 с `X-RateLimit-*` и `Retry-After`; 500 ошибка Redis; успешный путь до прокси; парсинг `extractAPIKey` |
| `internal/proxy/proxy_test.go` | unit | успешное проксирование (путь, тело, `Content-Type`); проброс статуса апстрима; недоступный апстрим → 502 |
| `internal/ratelimit/limiter_test.go` | unit | лимит 5; 6-й запрос блокируется; разные ключи не влияют друг на друга; `Limit()` |
| `internal/storage/postgres_test.go` | integration | поиск существующего ключа; отсутствующий ключ → nil; невалидный DSN → ошибка |

### Как запустить

```bash
# все тесты (интеграционные с PostgreSQL пропустятся без TEST_DB_DSN)
go test ./...

# подробный вывод по конкретному пакету
go test -v ./internal/handlers/

# один конкретный тест
go test -v -run TestGatewayRateLimitExceeded ./internal/handlers/

# с отчётом о покрытии
go test ./... -cover

# интеграционные тесты PostgreSQL (нужен живой Postgres, например из compose)
docker compose up -d postgres
TEST_DB_DSN="postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable" \
  go test -v ./internal/storage/
```

Тестовые зависимости: `github.com/alicebob/miniredis/v2` (in-memory Redis),
`net/http/httptest` и `testing` из стандартной библиотеки.

## Проверка качества кода

```bash
go build ./...      # сборка
go vet ./...        # статический анализ
gofmt -l .          # форматирование (пустой вывод = ок)
golangci-lint run ./...   # 30+ линтеров (errcheck, govet, ...)
staticcheck ./...   # статический анализ от Dominikh
```

## Документация кода (godoc)

Все экспортируемые пакеты, типы и функции снабжены godoc-комментариями
(`// Package ...`, `// Name ...` перед объявлением). Просмотреть документацию:

```bash
# по пакету
go doc ./internal/ratelimit

# по конкретному элементу
go doc ./internal/config.Config
go doc ./internal/ratelimit.RedisLimiter.Allow

# полный список экспорта пакета
go doc -all ./internal/storage
```

Локальный веб-браузер с документацией (опционально):

```bash
go install golang.org/x/tools/cmd/godoc@latest
godoc -http=:6060   # открыть http://localhost:6060/pkg/go-api-gateway/
```

## Схема БД

`migrations/schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL PRIMARY KEY,
    api_key    TEXT        NOT NULL UNIQUE,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO users (api_key, name) VALUES ('test-key-123', 'test user');
```

Счётчики rate limit в БД не хранятся — они живут в Redis с TTL 60 секунд
и очищаются сами.

## Особенности и ограничения

- Fixed window: в конце минуты счётчик обнуляется, возможен «рывок» на стыке окон.
- Лимит привязан к API-ключу, а не к IP.
- `stream: true` из Ollama пробрасывается корректно (`io.Copy`).
- При продакшене включи `sslmode` в `DB_DSN` и не логируй полные API-ключи.