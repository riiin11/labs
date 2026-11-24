# Trip Service

Сервис поездок: создание, получение и завершение поездки. HTTP API по OpenAPI,
данные в PostgreSQL, миграции goose, телеметрия через OpenTelemetry.

## Требования

- Go 1.24+
- `tripgoctl` — утилита курса для локального окружения
  (установка: [course-infra](https://github.com/course-go-autumn-2026/course-infra))
- `golangci-lint` — для `make lint`
- `make` (в Windows можно без него — команды из раздела «Команды» запускаются напрямую)

## Запуск

```bash
# 1. Окружение: PostgreSQL + OTel-коллектор
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect            # адреса Jaeger, Grafana, Prometheus

# 2. Миграции
export DATABASE_URL=$(grep '^DATABASE_URL=' .env | cut -d= -f2-)
make migrate

# 3. Сервис
make run
```

Адреса берутся из `.env`, который создаёт `tripgoctl environment start`.
В коде хосты и порты не хардкодятся.

## Переменные окружения

| Переменная | Обязательна | По умолчанию | Назначение |
|---|---|---|---|
| `HTTP_ADDR` | нет | `:8080` | адрес HTTP-сервера |
| `DATABASE_URL` | да | — | строка подключения к PostgreSQL |
| `LOG_LEVEL` | нет | `info` | уровень `slog` (`debug`, `info`, `warn`, `error`) |
| `SHUTDOWN_TIMEOUT` | нет | `10s` | сколько ждать активные запросы при остановке |
| `DATABASE_MAX_CONNS` | нет | `10` | максимум соединений в пуле |
| `DATABASE_MIN_CONNS` | нет | `1` | минимум соединений в пуле |
| `DATABASE_CONNECT_TIMEOUT` | нет | `5s` | таймаут подключения к БД |
| `DATABASE_QUERY_TIMEOUT` | нет | `3s` | таймаут запроса |
| `DATABASE_MAX_CONN_LIFETIME` | нет | `1h` | время жизни соединения |
| `OTEL_SERVICE_NAME` | нет | `trip-service` | имя сервиса в трейсах и метриках |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | нет | — | адрес OTLP-коллектора (`http://localhost:4318`) |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | нет | — | `http/protobuf` |
| `OTEL_RESOURCE_ATTRIBUTES` | нет | — | `service.namespace=tripgo,deployment.environment=local` |
| `OTEL_TRACES_SAMPLER` | нет | — | `parentbased_always_on` |

Пример — в `.env.example`. Реальные значения появятся в `.env` после
`tripgoctl environment start`; `.env` в репозиторий не коммитится.

## Команды

```bash
make generate           # oapi-codegen + go generate (моки)
make migrate            # goose up
make migrate-down       # goose down
make run                # go run ./cmd/trip-service
make lint               # golangci-lint run
make test               # go test -coverprofile coverage.out ./...
make test-integration   # go test -tags=integration ./internal/repository/...
```

Интеграционные тесты требуют поднятого PostgreSQL и применённых миграций:

```bash
tripgoctl environment start
make migrate
make test-integration
```

## API

| Метод | Путь | Ответы |
|---|---|---|
| `POST` | `/api/v1/trips` | `201`, `400`, `409`, `500` |
| `GET` | `/api/v1/trips/{tripId}` | `200`, `400`, `404`, `500` |
| `POST` | `/api/v1/trips/{tripId}/finish` | `200`, `400`, `404`, `409`, `500` |
| `GET` | `/health` | `200` |
| `GET` | `/ready` | `200`, `503` |

Ошибки — `application/problem+json` (RFC 9457) с полем `code`:
`invalid_request`, `trip_not_found`, `trip_completed`, `driver_busy`,
`internal_error`. Успешные ответы — `application/json`.

`/health` не ходит в БД и отвечает `200`, пока процесс жив. `/ready` проверяет
доступность БД и отдаёт `503`, если она недоступна. Обе ручки отдают
`application/json` со схемой `HealthResponse`, не `problem+json`.

## Архитектура

Слои и направления зависимостей — внутрь:

```
cmd/trip-service        main: конфиг, container.New, запуск, ожидание сигнала
internal/container      единственное место сборки зависимостей
internal/http           транспорт: хендлеры, маппинг ошибок, middleware
internal/usecase/trip   бизнес-логика: Create, Get, Finish
internal/domain/trip    модель, статусы, переходы, ошибки, валидация
internal/repository     SQL на squirrel, pgxpool
internal/infra          пул pgx, TxManager, telemetry
internal/generated      код, сгенерированный из OpenAPI
```

Правила:

- `internal/domain` не импортирует `net/http`, `chi`, `pgx`, `squirrel`, `otel`,
  `generated/api`;
- `internal/usecase` не импортирует `net/http`, `chi`, `pgx`, `squirrel`,
  `generated/api`;
- `internal/http/*` не импортирует `repository` и `infra`;
- `pgxpool.New` и конструкторы репозиториев — только в `container`;
- интерфейсы объявлены на стороне потребителя: `tripRepository` и `txManager` —
  в `internal/usecase/trip/contract.go`, `createUsecase`/`getUsecase`/`finishUsecase` —
  в `internal/http/trip/contract.go`. Моки генерируются `mockgen` в `mocks_test.go`
  рядом с кодом;
- маппинг «доменная ошибка → `Problem`» — в одном месте:
  `internal/http/domainerror`;
- middleware в порядке `Recover → Tracing → Logging → Metrics`.

**Отступление от п. 4.1.** В задании предложен пакет на действие
(`usecase/trip/create`, `http/trip/create` и т.д.). Здесь три действия лежат в
одном пакете `usecase/trip` и `http/trip`: на три ручки дробление на девять
пакетов даёт больше файлов, чем кода. Правила зависимостей и объявление
интерфейсов на стороне потребителя соблюдены. При появлении фоновых воркеров
(работа 3) и второго транспорта (работа 4) `http/trip` разделяется на пакеты
механически, потому что интерфейсы уже объявлены в `contract.go`.

## Менеджер транзакций

`TxManager.Do(ctx, fn)` открывает транзакцию с уровнем изоляции `ReadCommitted`,
кладёт её в `context` и вызывает `fn`. Если `fn` вернула `nil` — `COMMIT`, иначе
`ROLLBACK` и проброс ошибки; при панике — `ROLLBACK` и повторная паника.
Вложенный `Do` видит транзакцию в контексте и переиспользует её, вторую не
открывает. Репозиторий достаёт исполнителя из контекста
(`infra.TxFromContext`): есть транзакция — работает через неё, нет — через пул.
Транзакция не передаётся аргументом метода репозитория.

`ReadCommitted` выбран потому, что защита от гонок обеспечена не уровнем
изоляции, а ограничениями: частичный уникальный индекс
`trips (driver_id) WHERE status = 'active'` для запрета двух активных поездок и
условный `UPDATE ... WHERE status = 'active'` для завершения. `Serializable`
здесь не нужен и повышает риск сериализационных ошибок.

Создание поездки выполняется в одном `Do`: `INSERT` в `trips` и `INSERT` в
`trip_status_history`. При ошибке второй вставки первая откатывается —
это проверяет интеграционный `TestTxManager_Rollback`.

## Запрет двух активных поездок

Держится ограничением БД, а не проверкой в коде:

```sql
CREATE UNIQUE INDEX trips_one_active_per_driver
    ON trips (driver_id)
    WHERE status = 'active';
```

Два одновременных создания на одного водителя дают одну поездку и одну ошибку:
второй `INSERT` нарушает индекс, репозиторий распознаёт `23505` и возвращает
`trip.ErrDriverBusy`, хендлер отдаёт `409 driver_busy`.

## Observability

- Логи — `slog`, JSON, stdout, уровень из `LOG_LEVEL`. Одна запись на HTTP-запрос
  после ответа: `method`, `route` (шаблон, не путь с `tripId`), `status`,
  `duration_ms`, `trace_id`, `span_id`. `Info` для 2xx/4xx, `Error` для 5xx.
  Бизнес-события (`trip created`, `trip completed`) — в usecase, не в middleware.
- Трейсы — OTLP в коллектор, имя спана `{method} {http.route}`. Запросы в БД
  инструментированы `otelpgx` на уровне пула, поэтому каждый запрос — отдельный
  спан с `db.operation.name` и `db.collection.name`.
- Метрики — `http.server.request.duration`, `http.server.active_requests`,
  `db.client.operation.duration`, `tripgo.trips.created`,
  `tripgo.trips.completed`. В `http.route` только шаблон маршрута.
- При недоступном коллекторе сервис продолжает отвечать: телеметрия не
  критичная зависимость.

Проверка: `tripgoctl connect` → открыть Jaeger и Grafana, найти трейс запроса и
метрики `http_server_request_duration`, `tripgo_trips_created`.

## Тесты

| Что | Чем |
|---|---|
| Доменные правила (`Finish`, валидация) | table-driven, без моков |
| Usecase | `gomock`, моки через `go:generate` |
| Маппинг ошибок | table-driven |
| Хендлеры | `httptest` + мок usecase |
| Репозиторий | реальный PostgreSQL, `//go:build integration` |

Покрытие `internal/usecase/...` и `internal/domain/...` — не ниже 50%;
`make test` печатает итог. Интеграционные тесты изолированы: каждый готовит и
убирает свои данные и проходит при повторном запуске.