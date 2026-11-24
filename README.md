# Trip Service

Сервис поездок: создание, получение и завершение поездки, а также трекинг координат. HTTP API по OpenAPI, данные в PostgreSQL, миграции goose, телеметрия через OpenTelemetry, фоновый опрос устаревших поездок через worker pool.

## Требования

- Go 1.24+
- tripgoctl — утилита курса для локального окружения (установка: [course-infra](https://github.com/course-go-autumn-2026/course-infra))
- golangci-lint — для make lint
- make (в Windows можно без него — команды из раздела «Команды» запускаются напрямую)

## Запуск

```bash
# 1. Окружение: PostgreSQL + OTel-коллектор + Push Service
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect            # адреса Jaeger, Grafana, Prometheus, Push Service

# 2. Миграции
export DATABASE_URL=$(grep '^DATABASE_URL=' .env | cut -d= -f2-)
make migrate

# 3. Сервис
make run
```

Адреса берутся из `.env`, который создаёт `tripgoctl environment start`. В коде хосты и порты не хардкодятся.

## Переменные окружения

| Переменная | Обязательна | По умолчанию | Назначение |
| :--- | :---: | :--- | :--- |
| `HTTP_ADDR` | нет | `:8080` | адрес HTTP-сервера |
| `DATABASE_URL` | **да** | — | строка подключения к PostgreSQL |
| `LOG_LEVEL` | нет | `info` | уровень slog (`debug`, `info`, `warn`, `error`) |
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
| `POSITION_CHECK_INTERVAL` | нет | `30s` | период тика сканера устаревших поездок |
| `POSITION_STALE_AFTER` | нет | `60s` | через сколько секунд без координат поездка считается устаревшей |
| `POSITION_WORKERS` | нет | `4` | количество воркеров в пуле |
| `POSITION_QUEUE_SIZE` | нет | `100` | ёмкость очереди задач |
| `PUSH_HTTP_URL` | **да** (для ЛР3) | — | адрес Push Service из `.env` |
| `PUSH_REQUEST_TIMEOUT` | нет | `1s` | таймаут HTTP-запроса к Push Service |

Пример — в `.env.example`. Реальные значения появятся в `.env` после `tripgoctl environment start`; `.env` в репозиторий не коммитится.

## Команды

```bash
make generate           # oapi-codegen + go generate (моки)
make migrate            # goose up
make migrate-down       # goose down
make run                # go run ./cmd/trip-service
make lint               # golangci-lint run
make test               # go test -race -coverprofile coverage.out ./...
make test-integration   # go test -race -tags=integration ./internal/repository/...
```

Интеграционные тесты требуют поднятого PostgreSQL и применённых миграций:

```bash
tripgoctl environment start
make migrate
make test-integration
```

## API

| Метод | Путь | Ответы |
| :--- | :--- | :--- |
| `POST` | `/api/v1/trips` | `201`, `400`, `409`, `500` |
| `GET` | `/api/v1/trips/{tripId}` | `200`, `400`, `404`, `500` |
| `POST` | `/api/v1/trips/{tripId}/finish` | `200`, `400`, `404`, `409`, `500` |
| `POST` | `/api/v1/trips/{tripId}/positions` | `201`, `400`, `404`, `409`, `500` |
| `GET` | `/api/v1/trips/{tripId}/positions` | `200`, `400`, `404`, `500` |
| `GET` | `/health` | `200` |
| `GET` | `/ready` | `200`, `503` |

Ошибки — `application/problem+json` (RFC 9457) с полем `code`: `invalid_request`, `trip_not_found`, `trip_completed`, `driver_busy`, `internal_error`. Успешные ответы — `application/json`.

`/health` не ходит в БД и отвечает `200`, пока процесс жив. `/ready` проверяет доступность БД и отдаёт `503`, если она недоступна. Обе ручки отдают `application/json` со схемой `HealthResponse`, не `problem+json`.

## Архитектура

Слои и направления зависимостей — внутрь:

```text
cmd/trip-service          main: конфиг, container.New, запуск, ожидание сигнала
internal/container        единственное место сборки зависимостей
internal/http             транспорт: хендлеры, маппинг ошибок, middleware
internal/usecase/trip     бизнес-логика: Create, Get, Finish, SavePosition, GetPositions
internal/domain/trip      модель, статусы, переходы, ошибки, валидация
internal/repository       SQL на squirrel, pgxpool
internal/infra            пул pgx, TxManager, telemetry
internal/gateway/push     HTTP-клиент для Push Service
internal/worker/positions сканер, очередь, пул воркеров
internal/generated        код, сгенерированный из OpenAPI
```

Правила:

- `internal/domain` не импортирует `net/http`, `chi`, `pgx`, `squirrel`, `otel`, `generated/api`;
- `internal/usecase` не импортирует `net/http`, `chi`, `pgx`, `squirrel`, `generated/api`;
- `internal/http/*` не импортирует `repository` и `infra`;
- `pgxpool.New` и конструкторы репозиториев — только в `container`;
- интерфейсы объявлены на стороне потребителя: `tripRepository` и `txManager` — в `internal/usecase/trip/contract.go`, `createUsecase`/`getUsecase`/`finishUsecase` — в `internal/http/trip/contract.go`. Моки генерируются `mockgen` в `mocks_test.go` рядом с кодом;
- маппинг «доменная ошибка → Problem» — в одном месте: `internal/http/domainerror`;
- middleware в порядке `Recover → Metrics → Logging`.

## Менеджер транзакций

`TxManager.Do(ctx, fn)` открывает транзакцию с уровнем изоляции `ReadCommitted`, кладёт её в `context` и вызывает `fn`. Если `fn` вернула `nil` — `COMMIT`, иначе `ROLLBACK` и проброс ошибки; при панике — `ROLLBACK` и повторная паника. Вложенный `Do` видит транзакцию в контексте и переиспользует её, вторую не открывает. Репозиторий достаёт исполнителя из контекста (`infra.TxFromContext`): есть транзакция — работает через неё, нет — через пул.

`ReadCommitted` выбран потому, что защита от гонок обеспечена не уровнем изоляции, а ограничениями: частичный уникальный индекс для запрета двух активных поездок и условный `UPDATE ... WHERE status = 'active'` для завершения.

Создание поездки выполняется в одном `Do`: `INSERT` в `trips` и `INSERT` в `trip_status_history`. Сохранение координаты также выполняется в одной транзакции: `INSERT` в `trip_positions` и `UPDATE trips SET last_position_at = GREATEST(...)`. Обновление через `GREATEST` гарантирует, что опоздавшая координата не состарит поездку.

## Запрет двух активных поездок

Держится ограничением БД, а не проверкой в коде:

```sql
CREATE UNIQUE INDEX trips_one_active_per_driver
    ON trips (driver_id)
    WHERE status = 'active';
```

Два одновременных создания на одного водителя дают одну поездку и одну ошибку: второй `INSERT` нарушает индекс, репозиторий распознаёт `23505` и возвращает `trip.ErrDriverBusy`, хендлер отдаёт `409 driver_busy`.

## Опрос устаревших поездок (ЛР3)

Для поддержания актуальности координат в активных поездках сервис использует фоновый опрос устройств через Push Service.

### Архитектура опроса

Схема работы: **Сканер (тикер) → Ограниченная очередь → Пул воркеров → Push Service**

1. **Сканер** (одна фоновая горутина) раз в `POSITION_CHECK_INTERVAL` делает запрос в БД и достает поездки, которые не присылали координаты дольше `POSITION_STALE_AFTER`.
2. **Очередь** — буферизованный канал (`POSITION_QUEUE_SIZE`). Внутри реализована дедупликация: одна поездка не может находиться в очереди дважды.
3. **Воркеры** (`POSITION_WORKERS` штук) разбирают очередь и делают HTTP-запрос в Push Service с таймаутом `PUSH_REQUEST_TIMEOUT`.
4. **Отмена:** если во время нахождения поездки в очереди приходит новая координата, задача отменяется и в Push Service не уходит.

### Поведение при переполнении очереди

Сканер не блокируется на записи в очередь. Если очередь заполнена:

- поездка отбрасывается (дропается);
- инкрементируется метрика `tripgo.stale_checks{result="dropped"}`;
- пишется лог уровня `Warn`.

Поездка при этом не теряется: она по-прежнему остается без свежих координат и попадет в выборку на следующем тике сканера. Повторный пуш уйдет через `POSITION_CHECK_INTERVAL`.

### Оптимизация БД

Для запроса выборки устаревших поездок создан частичный функциональный индекс:

```sql
CREATE INDEX idx_trips_stale_check 
ON trips (status, COALESCE(last_position_at, started_at)) 
WHERE status = 'active';
```

Индекс проверен через `EXPLAIN (ANALYZE)` — планировщик использует `Index Scan` (время выполнения ~2 мс на 5000 строк).

### Graceful Shutdown

Порядок остановки компонентов строго соблюдается (п. 5.7):

1. HTTP-сервер перестает принимать новые запросы.
2. Сканер останавливается (новые задачи не появляются).
3. Воркеры дорабатывают оставшиеся в очереди задачи.
4. Закрываются пул БД и экспортеры OpenTelemetry.

## Observability

- **Логи** — `slog`, JSON, stdout, уровень из `LOG_LEVEL`. Одна запись на HTTP-запрос после ответа: `method`, `route`, `status`, `duration_ms`, `trace_id`, `span_id`.
- **Трейсы** — OTLP в коллектор. Вызов Push Service в воркере создает отдельный спан `push.request_position`.
- **Метрики** — `http.server.request.duration`, `http.server.active_requests`, `tripgo.trips.created`, `tripgo.trips.completed`, `tripgo.positions.saved`, `tripgo.stale_checks`, `tripgo.position_requests.queue.size`, `tripgo.position_requests.workers.active`.