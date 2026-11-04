# ТЗ: HR-система (учёт сотрудников)

> Обновлено: по указанию ментора refresh-токены хранятся в Redis, таблицы `refresh_tokens` нет.

Oct 25, 2026 · @SHEFF

REST API на Go в структуре demo-project: HR управляет сотрудниками, отделами и должностями, сотрудник видит только свой профиль. Строим по шагам, каждый шаг проверяем в Postman.

## Стек и структура

Стек: Go, Gin (`github.com/gin-gonic/gin`; сервер остаётся стандартным `http.Server`, Gin только роутер), PostgreSQL 17 (pgx/v5), JWT (golang-jwt), bcrypt, slog, godotenv + envconfig. Redis 7 (`go-redis/v9`): хранит refresh-токены (так сказал ментор), в таблицу БД они не попадают. В Docker два сервиса: Postgres на порту **5433** (5432 занят локальным Postgres) и Redis на порту **6380** (6379 занят локально).

```
hr-system/
  cmd/api/main.go
  internal/
    config/  app/  models/
    repository/  user role refresh department position employee salary timeoff
    service/     auth department position employee salary timeoff
    handler/     v1  middleware  response
  pkg/postgres  pkg/redis  pkg/logger
  migrations/  docs/openapi.yaml
  docker-compose.infra.yaml  Makefile  .env.example  README.md
```

Слои: handler → service → repository → models. Интерфейсы лежат в `service/service.go` и `repository/repository.go`. Ошибки в `models/errors.go`: ErrNotFound, ErrInvalidInput, ErrConflict, ErrForbidden, ErrInvalidToken, ErrInvalidCredentials.

## HTTP-слой на Gin (решение ментора)

HTTP-слой строим на **Gin** (`github.com/gin-gonic/gin`, версия как в примере ментора: v1.12.0), а не на `net/http` ServeMux. Всё остальное в ТЗ (схема БД, роли, API, коды ответов, шаги) не меняется. Раздел заменяет отдельный файл `TZ-gin-addendum.md`: всё из него теперь здесь, а упоминания `net/http`, `mux.Handle` и `w, r` выше уже приведены к Gin.

**Что не меняется:** `models`, `repository`, `service`, `pkg/*`, `internal/config`, миграции, Docker, Makefile, `docs/openapi.yaml`, тесты сервисов. Правило зависимостей прежнее: handler → service → repository → models. Gin импортируется только в `handler` и в `app.go` (для `gin.SetMode`).

`http.Server` в `app.go` остаётся стандартным: таймауты `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_SHUTDOWN_TIMEOUT` и graceful shutdown работают как раньше. `*gin.Engine` реализует `http.Handler`, поэтому его передают в `http.Server{Handler: engine}`.

### Таблица замен

| Что делаем | `net/http` | Gin |
|---|---|---|
| Сигнатура обработчика | `func(w http.ResponseWriter, r *http.Request)` | `func(c *gin.Context)` |
| Маршрут | `mux.Handle("GET /api/v1/departments", h)` | `group.GET("/departments", h)` |
| Параметр пути `{id}` | `r.PathValue("id")` | `c.Param("id")` (в маршруте `/:id`) |
| Query `?q=...` | `r.URL.Query().Get("q")` | `c.Query("q")`, `c.DefaultQuery("page", "1")` |
| Тело JSON | `json.NewDecoder(r.Body).Decode(&v)` | `c.ShouldBindJSON(&v)` |
| Ответ JSON | `response.JSON(w, 200, v)` | `c.JSON(200, v)` |
| Ошибка | `response.Error(w, 400, "msg")` | `c.AbortWithStatusJSON(400, gin.H{"error": "msg"})` |
| Ответ без тела (204) | `w.WriteHeader(204)` | `c.Status(204)` |
| Контекст запроса | `r.Context()` | `c.Request.Context()` |
| Данные от middleware к handler | `context.WithValue` | `c.Set("claims", ...)` и `c.Get("claims")` |
| Заголовок ответа | `w.Header().Set(...)` | `c.Header(...)` |
| Файл из формы (импорт CSV) | `r.FormFile("file")` | `c.FormFile("file")` |
| Писать в ответ (экспорт CSV) | `csv.NewWriter(w)` | `csv.NewWriter(c.Writer)` |
| Middleware | `func(next http.Handler) http.Handler` | `gin.HandlerFunc`, внутри `c.Next()` и `c.Abort()` |

### Правила, которые остаются в силе

- **Валидация только в `Validate()` у входных структур в `models`.** Теги `binding:"required,email"` Gin не используем: в ТЗ запрещены библиотеки валидации. `c.ShouldBindJSON` нужен только чтобы разобрать JSON в структуру DTO. Ошибка формата JSON даёт 400.
- **Запрет лишних полей в теле:** один раз при старте `binding.EnableDecoderDisallowUnknownFields = true` (пакет `github.com/gin-gonic/gin/binding`), в `InitRoutes()`.
- **Лимит тела запроса:** перед разбором `c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)` (для импорта CSV тот же 1 МБ).
- **Ошибки логирует только `response.go`:** 4xx как Warn, 500 как Error, с `c.Request.Context()`. Своих `log.Printf` и `slog` в handler не пишем.
- **`ServiceError(c, err)`** переводит `models.Err...` в коды (NotFound 404, InvalidInput 400, Conflict 409, Forbidden 403, InvalidToken 401, InvalidCredentials 401, остальное 500).
- **`requireHR`** остаётся локальной переменной в `InitRoutes()`: цепочка `Auth` + `RequireRole("hr")` (403).
- **`CurrentUser`:** handler собирает её из данных токена в `gin.Context` и передаёт сервису вторым параметром. Сервис про Gin ничего не знает.
- Ответы и ошибки на английском, формат `{"error":"текст"}`. Id в DTO `int64`, JSON в `snake_case`.

### Особенности Gin

- Движок создаём через `gin.New()`, а не `gin.Default()`: свои `RequestID`, `Logging`, `Recover`, чтобы логи шли через `slog`.
- Порядок middleware: `RequestID` → `CORS` → `Logging` → `Recover`, подключаются `router.Use(...)`.
- В `prod` включаем `gin.SetMode(gin.ReleaseMode)` (`APP_ENV=prod`).
- **Конфликт `/employees/export.csv` и `/employees/:id`:** оба маршрута на одном уровне. В современных версиях Gin это разрешено (статический сегмент приоритетнее параметра), но проверить при шаге 12.
- `GET /health` регистрируется на корне (`router.GET("/health", ...)`), вне `/api/v1`.
- CORS для Swagger UI (`http://localhost:8081`): собственный middleware.
- Для тестов handler при желании `httptest` и `gin.SetMode(gin.TestMode)`; тесты сервисов от Gin не зависят.

### Вопросы к ментору по Gin

- Свои `Logging` и `Recover` под `slog` (как в ТЗ) или готовые `gin.Logger()` и `gin.Recovery()`?
- Оставляем валидацию в `Validate()` без тегов `binding`?

## Роли и права

| Действие                                                                                             | HR                                    | Сотрудник                    |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------- | ------------------------------------- |
| Login, refresh, logout                                                                                       | да                                  | да                                  |
| Свой профиль (`GET /me`, с отделом и должностью)                             | да                                  | да                                  |
| Сотрудники: список, просмотр, создание, изменение, удаление | да                                  | нет                                |
| Отделы и должности: все операции                                                  | да                                  | нет                                |
| Зарплаты (этап 2)                                                                                | все                                | только свои, чтение   |
| Отпуска и больничные (этап 2)                                                          | одобряет и видит все | создаёт и видит свои |

Публичной регистрации нет. Аккаунт создаёт HR: создание сотрудника = запись в `users` и `employees` в одной транзакции. Первого HR создаёт seed-миграция.

## Схема БД: таблицы, поля, связи

Всего 8 таблиц, 5 нужны для этапа 1 (roles, users, departments, positions, employees), `salaries` и `time_off` для этапа 2, `employee_position_history` для этапа 3. Типы PostgreSQL. Во всех таблицах `created_at` (и `updated_at`, где данные меняются) типа TIMESTAMPTZ. Refresh-токены хранятся не в БД, а в Redis (см. ниже). Готовый SQL лежит в шагах плана (миграции).

### Связи

| Связь                                | Тип                     | Как задана                                                   |
| ----------------------------------------- | -------------------------- | --------------------------------------------------------------------- |
| roles → users                            | один ко многим | `users.role_id` → `roles.id`                                     |
| users → employees                        | один к одному   | `employees.user_id` → `users.id`, UNIQUE                         |
| departments → employees                  | один ко многим | `employees.department_id` → `departments.id`, ON DELETE RESTRICT |
| positions → employees                    | один ко многим | `employees.position_id` → `positions.id`, ON DELETE RESTRICT     |
| employees → salaries                     | один ко многим | `salaries.employee_id` → `employees.id`, ON DELETE CASCADE       |
| employees → time_off                     | один ко многим | `time_off.employee_id` → `employees.id`, ON DELETE CASCADE       |
| users → time_off (кто одобрил) | один ко многим | `time_off.approved_by` → `users.id`, может быть NULL    |
| employees → employee_position_history    | один ко многим | `employee_position_history.employee_id` → `employees.id`         |

### roles (справочник ролей)

| Поле   | Тип      | Ограничения  | Зачем                                                                  |
| ---------- | ----------- | ----------------------- | --------------------------------------------------------------------------- |
| id         | SERIAL      | PK                      | идентификатор                                                  |
| code       | VARCHAR(20) | NOT NULL, UNIQUE        | `hr` или `employee`: по нему проверяются права |
| name       | VARCHAR(50) | NOT NULL                | название для людей                                          |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | служебное                                                          |

### users (аккаунты для входа)

| Поле               | Тип       | Ограничения                       | Зачем                                                |
| ---------------------- | ------------ | -------------------------------------------- | --------------------------------------------------------- |
| id                     | SERIAL       | PK                                           | идентификатор                                |
| role_id                | INT          | NOT NULL, FK → roles                        | роль аккаунта                                 |
| email                  | VARCHAR(100) | NOT NULL, UNIQUE, CHECK email = lower(email) | логин, всегда в нижнем регистре |
| password_hash          | VARCHAR(255) | NOT NULL                                     | bcrypt-хэш пароля                                |
| is_active              | BOOLEAN      | NOT NULL, DEFAULT true                       | false = вход запрещён (уволен)          |
| created_at, updated_at | TIMESTAMPTZ  | NOT NULL, DEFAULT now()                      | служебное                                        |

### Refresh-токены (Redis, не таблица)

| Ключ в Redis                              | Значение                                          | TTL                               | Зачем                                                                                       |
| ---------------------------------------------- | --------------------------------------------------------- | --------------------------------- | ------------------------------------------------------------------------------------------------ |
| `refresh:<SHA-256 хэш токена>`      | id пользователя                               | 30 дней (`JWT_REFRESH_TTL`) | сам токен не хранится, только его хэш                              |
| `refresh_user:<id пользователя>` | множество хэшей его токенов (SET) | 30 дней                       | чтобы отозвать все токены сотрудника при увольнении |

Токен действует, пока ключ `refresh:<хэш>` существует. Logout и обновление удаляют ключ.

### departments (отделы)

| Поле               | Тип       | Ограничения  | Зачем                    |
| ---------------------- | ------------ | ----------------------- | ----------------------------- |
| id                     | SERIAL       | PK                      | идентификатор    |
| name                   | VARCHAR(100) | NOT NULL, UNIQUE        | название отдела |
| created_at, updated_at | TIMESTAMPTZ  | NOT NULL, DEFAULT now() | служебное            |

### positions (должности)

| Поле               | Тип        | Ограничения  | Зачем                                                           |
| ---------------------- | ------------- | ----------------------- | -------------------------------------------------------------------- |
| id                     | SERIAL        | PK                      | идентификатор                                           |
| name                   | VARCHAR(100)  | NOT NULL, UNIQUE        | название должности                                  |
| base_salary            | NUMERIC(12,2) | NOT NULL, CHECK >= 0    | базовый оклад по должности (ориентир) |
| created_at, updated_at | TIMESTAMPTZ   | NOT NULL, DEFAULT now() | служебное                                                   |

### employees (сотрудники)

| Поле               | Тип      | Ограничения                                                            | Зачем                                                          |
| ---------------------- | ----------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| id                     | SERIAL      | PK                                                                                | идентификатор                                          |
| user_id                | INT         | NOT NULL, UNIQUE, FK → users                                                     | аккаунт сотрудника (email берём отсюда) |
| department_id          | INT         | NOT NULL, FK → departments, RESTRICT, индекс                               | отдел                                                          |
| position_id            | INT         | NOT NULL, FK → positions, RESTRICT, индекс                                 | должность                                                  |
| first_name             | VARCHAR(50) | NOT NULL                                                                          | имя                                                              |
| last_name              | VARCHAR(50) | NOT NULL                                                                          | фамилия                                                      |
| phone                  | VARCHAR(20) | NOT NULL                                                                          | телефон (обязателен)                               |
| hire_date              | DATE        | NOT NULL                                                                          | дата приёма                                               |
| termination_date       | DATE        | NULL, CHECK >= hire_date                                                          | дата увольнения                                       |
| status                 | VARCHAR(20) | NOT NULL, DEFAULT 'active', CHECK IN (active, on_leave, terminated), индекс | состояние                                                  |
| created_at, updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now()                                                           | служебное                                                  |

### salaries (история зарплат, этап 2)

| Поле       | Тип        | Ограничения                       | Зачем                                                                                                                                                       |
| -------------- | ------------- | -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id             | SERIAL        | PK                                           | идентификатор                                                                                                                                       |
| employee_id    | INT           | NOT NULL, FK → employees, CASCADE           | чья зарплата                                                                                                                                          |
| amount         | NUMERIC(12,2) | NOT NULL, CHECK > 0                          | сумма                                                                                                                                                       |
| effective_date | DATE          | NOT NULL, UNIQUE вместе с employee_id | с какой даты действует. Текущая зарплата = строка с самой поздней датой не позже сегодня |
| created_at     | TIMESTAMPTZ   | NOT NULL, DEFAULT now()                      | служебное                                                                                                                                               |

### time_off (отпуска и больничные, этап 2)

| Поле               | Тип      | Ограничения                                              | Зачем                                                                    |
| ---------------------- | ----------- | ------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| id                     | SERIAL      | PK                                                                  | идентификатор                                                    |
| employee_id            | INT         | NOT NULL, FK → employees, CASCADE, индекс                    | кто просит                                                           |
| approved_by            | INT         | NULL, FK → users                                                   | кто из HR одобрил или отклонил (NULL пока pending) |
| leave_type             | VARCHAR(20) | NOT NULL, CHECK IN (vacation, sick)                                 | отпуск или больничный                                      |
| start_date, end_date   | DATE        | NOT NULL, CHECK end_date >= start_date                              | период                                                                  |
| status                 | VARCHAR(20) | NOT NULL, DEFAULT 'pending', CHECK IN (pending, approved, rejected) | состояние заявки                                               |
| created_at, updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now()                                             | служебное                                                            |

### employee_position_history (история перемещений, этап 3)

| Поле      | Тип      | Ограничения                           | Зачем                                                       |
| ------------- | ----------- | ------------------------------------------------ | ---------------------------------------------------------------- |
| id            | SERIAL      | PK                                               | идентификатор                                       |
| employee_id   | INT         | NOT NULL, FK → employees, CASCADE, индекс | чья история                                            |
| department_id | INT         | NOT NULL, FK → departments                      | отдел в этот период                              |
| position_id   | INT         | NOT NULL, FK → positions                        | должность в этот период                      |
| start_date    | DATE        | NOT NULL                                         | с какой даты                                           |
| end_date      | DATE        | NULL, CHECK >= start_date                        | по какую дату, NULL = действует сейчас |
| created_at    | TIMESTAMPTZ | NOT NULL, DEFAULT now()                          | служебное                                               |

## Файлы и папки: где что создать

Корень проекта: `~/Desktop/hr-system` (рядом с `demo-project2`). Все пути ниже относительно корня. Пакет Go = одна папка, имя пакета равно имени папки (`package department`). Импорты с начала модуля: `hr-system/internal/models`.

### Корень

| Файл                      | Для чего                                              | Что внутри                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| ----------------------------- | ------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go.mod`                    | модуль и зависимости                       | создаётся `go mod init hr-system`                                                                                                                                                                                                                                                                                                                                                                                                          |
| `.gitignore`                | не коммитить лишнее                         | `.env`, `*.log`, `.idea/`                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `.env.example`              | шаблон настроек (в git)                       | `APP_ENV`, `LOG_LEVEL`, `LOG_FILE`, `HTTP_PORT=8080`, `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_SHUTDOWN_TIMEOUT`, `POSTGRES_HOST=localhost`, `POSTGRES_PORT=5433`, `POSTGRES_USER=hr`, `POSTGRES_PASSWORD=hr`, `POSTGRES_DB=hr-db`, `POSTGRES_SSLMODE=disable`, `REDIS_HOST=localhost`, `REDIS_PORT=6380`, `REDIS_PASSWORD`, `REDIS_DB=0`, `JWT_SECRET`, `JWT_ACCESS_TTL=15m`, `JWT_REFRESH_TTL=720h` |
| `.env`                      | реальные настройки (не в git)            | копия `.env.example`: `cp .env.example .env`                                                                                                                                                                                                                                                                                                                                                                                                 |
| `docker-compose.infra.yaml` | поднять Postgres и Redis одной командой | сервис `postgres` (образ `postgres:17-alpine`, контейнер `hr-postgres`, порт `127.0.0.1:5433:5432`, volume `pgdata`, монтирование `./migrations` в `/docker-entrypoint-initdb.d`, healthcheck `pg_isready`) и сервис `redis` (образ `redis:7-alpine`, контейнер `hr-redis`, порт `127.0.0.1:6380:6379`, healthcheck `redis-cli ping`)                         |
| `Makefile`                  | короткие команды                              | `infra-up`, `infra-down`, `infra-reset`, `run`                                                                                                                                                                                                                                                                                                                                                                                                |
| `README.md`                 | как запустить                                    | команды запуска, ссылка на `docs/`                                                                                                                                                                                                                                                                                                                                                                                            |

### cmd/ и pkg/

| Файл                     | Для чего                 | Что внутри                                                                                                                                 |
| ---------------------------- | ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/api/main.go`          | точка входа           | загрузить config, создать logger, вызвать `app.Run(cfg)`, при ошибке записать в лог и `os.Exit(1)` |
| `pkg/logger/logger.go`     | настройка `slog`     | `New(level, file)`: в `local` пишет в файл `app.log`, иначе в stdout                                                         |
| `pkg/postgres/postgres.go` | подключение к БД  | `NewPool(ctx, cfg) (*pgxpool.Pool, error)`: создать pool и сделать `Ping`                                                        |
| `pkg/redis/redis.go`       | подключение к Redis | `NewClient(ctx, cfg) (*redis.Client, error)`: создать клиент и сделать `Ping`                                              |

### internal/config, internal/app, internal/models

| Файл                      | Для чего                                                                 | Что внутри                                                                                                                              |
| ----------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/config/config.go` | настройки из `.env`                                                | структура `Config` (HTTP, Postgres, JWT, Log) с тегами `envconfig`, `Load()` читает `.env` через `godotenv` |
| `internal/app/app.go`       | сборка всего приложения                                    | поднять pool → создать repositories → services → handler → `http.Server`; запуск и graceful shutdown по `Ctrl+C`  |
| `internal/models/*.go`      | доменные структуры, не знают ни о БД, ни о HTTP | `user.go`, `role.go`, `department.go`, `position.go`, `employee.go`, `salary.go`, `timeoff.go`, `errors.go`                      |
| `internal/models/errors.go` | общие ошибки                                                         | `ErrNotFound`, `ErrInvalidInput`, `ErrConflict`, `ErrForbidden`, `ErrInvalidToken`, `ErrInvalidCredentials`                          |

### internal/repository (работа с БД, только SQL)

| Файл                                        | Для чего                                | Что внутри                                                                                                                                                                                                                                                                                                                  |
| ----------------------------------------------- | ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `repository/repository.go`                    | контракты                             | интерфейсы `UserRepository`, `RoleRepository`, `RefreshTokenRepository`, `DepartmentRepository`, `PositionRepository`, `EmployeeRepository`, (позже `SalaryRepository`, `TimeOffRepository`) и структура `Repositories`                                                                   |
| `repository/<сущность>/repository.go` | реализация для pgx                | папки `user`, `role`, `refresh`, `department`, `position`, `employee`, `salary`, `timeoff`; `NewRepository(db *pgxpool.Pool)`, методы с SQL; `pgx.ErrNoRows` → `models.ErrNotFound`, код `23505` (дубликат) и `23503` (связанные записи) → `models.ErrConflict` |
| `repository/<сущность>/db_model.go`   | структура строки таблицы | `dbModel` с тегами `db:"..."` и метод `toModel()` (как в demo-project)                                                                                                                                                                                                                                        |

### internal/service (бизнес-логика)

| Файл                                              | Для чего              | Что внутри                                                                                                                                                                                            |
| ----------------------------------------------------- | ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `service/service.go`                                | контракты           | интерфейсы `AuthService`, `DepartmentService`, `PositionService`, `EmployeeService` (позже `SalaryService`, `TimeOffService`, `ReportService`) и структура `Services` |
| `service/auth/`                                     | вход и токены     | `service.go` (структура, `NewService`), `login.go`, `refresh.go`, `logout.go`, `tokens.go` (выпуск JWT и refresh), `authenticate.go` (проверка access-токена)      |
| `service/department/`, `position/`, `employee/` | CRUD с проверками | `service.go` + по файлу на операцию (`create.go`, `get.go`, `list.go`, `update.go`, `delete.go`); валидация (trim, длины, обязательные поля)        |

### internal/handler (HTTP)

| Файл                                                                               | Для чего                          | Что внутри                                                                                                                                                                                                                                               |
| -------------------------------------------------------------------------------------- | ---------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `handler/handler.go` | маршруты | `InitRoutes() *gin.Engine`: `gin.New()`, группа `/api/v1`, маршруты вида `group.GET("/departments", h)`, защита `requireAuth` и `requireHR`, общая цепочка `router.Use(RequestID, CORS, Logging, Recover)`; `GET /health` на корне |
| `handler/v1/handler.go`                                                              | структура v1                    | `Handler{services}`, `NewHandler`                                                                                                                                                                                                                             |
| `handler/v1/request.go` | общие помощники | `decodeJSON` (`c.ShouldBindJSON`, лимит тела 1 МБ, запрет лишних полей), `parseID` (`c.Param("id")`), `parseDate`, `currentUser` |
| `handler/v1/auth.go`, `me.go`, `department.go`, `position.go`, `employee.go` | обработчики                   | разобрать запрос → вызвать сервис → отдать JSON; без бизнес-логики                                                                                                                                             |
| `handler/v1/dto.go`                                                                  | формат JSON                        | структуры запросов и ответов с тегами `json:"first_name"` (snake_case)                                                                                                                                                          |
| `handler/middleware/auth.go` | защита | `Auth` (читает `Authorization: Bearer`, кладёт данные токена в `gin.Context`), `RequireRole("hr")` (403), `ClaimsFromContext`; все как `gin.HandlerFunc` |
| `handler/middleware/request_id.go`, `logging.go`, `recover.go`, `cors.go` | общее | id запроса в заголовке `X-Request-ID`, лог запроса (метод, путь, статус, время), перехват паники → 500, разрешение для Swagger UI (`localhost:8081`) |
| `handler/response/response.go` | единый формат ответов | `JSON(c, status, v)`, `Error(c, status, msg)` → `{"error":"текст"}` (через `c.AbortWithStatusJSON`), `ServiceError(c, err)` переводит ошибки в коды: NotFound 404, InvalidInput 400, Conflict 409, Forbidden 403, InvalidToken 401, InvalidCredentials 401, остальное 500 |

### migrations/ и docs/

| Файл                                                                            | Для чего                           | Что внутри                                                                                    |
| ----------------------------------------------------------------------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `migrations/001_auth.sql`                                                         | таблицы входа                 | `roles`, `users` + seed ролей                                            |
| `migrations/002_departments_positions.sql`                                        | справочники                    | `departments`, `positions`                                                                         |
| `migrations/003_employees.sql`                                                    | сотрудники                      | `employees`                                                                                          |
| `migrations/004_salaries.sql`, `005_time_off.sql`, `006_position_history.sql` | этапы 2 и 3                         | соответствующие таблицы                                                          |
| `docs/openapi.yaml`                                                               | описание API                      | все эндпоинты, поля, коды ответов                                           |
| `swagger-ui/docker-compose.yaml`                                                  | просмотр документации | образ `swaggerapi/swagger-ui`, порт 8081, монтирование `../docs/openapi.yaml` |

**Правило зависимостей:** handler → service → repository → models. Нижний слой никогда не импортирует верхний (repository не знает о handler).

## API

Префикс `/api/v1`. Коды: 200, 201, 204, 400, 401, 403, 404, 409.

| Метод и путь                                                                                          | Кто                                      | Этап |
| --------------------------------------------------------------------------------------------------------------- | ------------------------------------------- | -------- |
| `POST /auth/login`, `/auth/refresh`, `/auth/logout`                                                       | публично (refresh в теле)      | 1        |
| `GET /me`                                                                                                     | любая роль                         | 1        |
| `GET/POST /departments`, `GET/PUT/DELETE /departments/{id}` (409, если есть сотрудники)   | HR                                          | 1        |
| `GET/POST /positions`, `GET/PUT/DELETE /positions/{id}`                                                     | HR                                          | 1        |
| `GET/POST /employees`, `GET/PUT/DELETE /employees/{id}`                                                     | HR                                          | 1        |
| `GET /employees?q=&department_id=&position_id=&status=&page=&limit=`                                          | HR                                          | 1        |
| `POST/GET /employees/{id}/salaries`, `GET /me/salaries`                                                     | HR / сотрудник (свои)          | 2        |
| `GET /reports/departments` (число сотрудников, сумма и средняя зарплата) | HR                                          | 2        |
| `POST /time-off`, `GET /time-off`                                                                           | сотрудник (свои) / HR (все) | 2        |
| `PATCH /time-off/{id}/status` (approved / rejected)                                                           | HR                                          | 2        |
| `GET /employees/{id}/history`                                                                                 | HR                                          | 3        |
| `GET /employees/export.csv`, `POST /employees/import`                                                       | HR                                          | 3        |

## Пошаговый план: от пустой папки до готового API

12 шагов. Каждый шаг: что создать → что в нём написать → как проверить → когда шаг закончен. В ТЗ нет кода Go: полный рабочий код каждого шага пишется в чате по этому плану. Дальше не идём, пока проверка шага не прошла.

### Шаг 1. Каркас проекта и /health

**Цель:** пустое приложение запускается, подключается к БД и отвечает на `GET /health`.

**Команды** (Git Bash):

```
cd ~/Desktop
mkdir hr-system && cd hr-system
go mod init hr-system
mkdir -p cmd/api internal/config internal/app internal/models internal/repository internal/service internal/handler/v1 internal/handler/middleware internal/handler/response pkg/postgres pkg/logger migrations docs
git init
```

**Файлы в порядке создания:**

1. `.gitignore`, `.env.example`, `.env` (копия примера), содержимое — в таблице «Корень» выше.
2. `docker-compose.infra.yaml`:

```
services:
  postgres:
    image: postgres:17-alpine
    container_name: hr-postgres
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-hr}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-hr}
      POSTGRES_DB: ${POSTGRES_DB:-hr-db}
    ports:
      - "127.0.0.1:5433:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB}"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
  redis:
    image: redis:7-alpine
    container_name: hr-redis
    ports:
      - "127.0.0.1:6380:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 10
volumes:
  pgdata:
```

3. `Makefile` (отступ перед командами — символ Tab):

```
COMPOSE := docker compose -f docker-compose.infra.yaml
.PHONY: run infra-up infra-down infra-reset
run:
	go run ./cmd/api
infra-up:
	$(COMPOSE) up -d --wait
infra-down:
	$(COMPOSE) down
infra-reset:
	$(COMPOSE) down -v
	$(COMPOSE) up -d --wait
```

4. `pkg/logger/logger.go`, `pkg/postgres/postgres.go`, `pkg/redis/redis.go`, `internal/config/config.go`.
5. `internal/handler/v1/health.go` (отдаёт `{"status":"ok"}`, 503 если `Ping` Postgres или Redis не прошёл) и `internal/handler/handler.go` (маршрут `GET /health`).
6. `internal/app/app.go` (pool → handler → server → graceful shutdown), `cmd/api/main.go`.
7. В конце: `go mod tidy`.

**Проверка:** `make infra-up` (или `docker compose -f docker-compose.infra.yaml up -d --wait`), `make run`, в `app.log` строка `server started`. В Postman создать коллекцию `hr-system` с переменной `baseUrl = http://localhost:8080`, `GET {{baseUrl}}/health` → **200**.

**Готово, когда:** сервер запускается, `Ctrl+C` останавливает его чисто (в логе `shutting down`), `/health` даёт 200.

### Шаг 2. Миграции этапа 1

**Цель:** в БД есть 5 таблиц и данные: роли `hr`, `employee`. Первого HR-аккаунта в миграции нет: его создаёт `go run ./cmd/seed` (пароль вводится с клавиатуры и в репозиторий не попадает).

**Файл **`migrations/001_auth.sql`**:**

```
CREATE TABLE roles (
    id         SERIAL PRIMARY KEY,
    code       VARCHAR(20) NOT NULL UNIQUE,
    name       VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    role_id       INT NOT NULL REFERENCES roles(id),
    email         VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_email_lower CHECK (email = lower(email))
);
CREATE INDEX idx_users_role_id ON users(role_id);

INSERT INTO roles (code, name) VALUES ('hr', 'HR'), ('employee', 'Сотрудник');

-- первого HR здесь нет: его создаёт go run ./cmd/seed (пароль не хранится в репозитории)
```

**Файл **`migrations/002_departments_positions.sql`**:**

```
CREATE TABLE departments (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE positions (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    base_salary NUMERIC(12,2) NOT NULL CHECK (base_salary >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Файл **`migrations/003_employees.sql`**:**

```
CREATE TABLE employees (
    id               SERIAL PRIMARY KEY,
    user_id          INT NOT NULL UNIQUE REFERENCES users(id),
    department_id    INT NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
    position_id      INT NOT NULL REFERENCES positions(id) ON DELETE RESTRICT,
    first_name       VARCHAR(50) NOT NULL,
    last_name        VARCHAR(50) NOT NULL,
    phone            VARCHAR(20) NOT NULL,
    hire_date        DATE NOT NULL,
    termination_date DATE,
    status           VARCHAR(20) NOT NULL DEFAULT 'active'
                     CHECK (status IN ('active', 'on_leave', 'terminated')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT employees_dates CHECK (termination_date IS NULL OR termination_date >= hire_date)
);
CREATE INDEX idx_employees_department_id ON employees(department_id);
CREATE INDEX idx_employees_position_id ON employees(position_id);
CREATE INDEX idx_employees_status ON employees(status);
CREATE INDEX idx_employees_name ON employees(last_name, first_name);
```

**Применение:** файлы из `migrations/` выполняются по алфавиту только при первом создании базы. Изменили или добавили миграцию → `make infra-reset` (удалит данные и применит заново).

**Проверка:**

```
make infra-reset
docker exec hr-postgres psql -U hr -d hr-db -c "\dt"
docker exec hr-postgres psql -U hr -d hr-db -c "SELECT code FROM roles;"
docker exec hr-postgres psql -U hr -d hr-db -c "SELECT email FROM users;"
```

**Готово, когда:** `\dt` показывает 5 таблиц, в `roles` две строки (`hr`, `employee`), `users` пока пустая (первого HR создаёт `go run ./cmd/seed`).

### Шаг 3. Авторизация (login, refresh, logout, роли)

**Цель:** HR входит по email и паролю, получает пару токенов, защищённые маршруты проверяют токен и роль.

**Файлы (в порядке создания):**

| Файл                                                        | Что написать                                                                                                                                                                                                                                                              |
| --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/models/errors.go`                                   | все `Err...` (см. таблицу выше)                                                                                                                                                                                                                                    |
| `internal/models/user.go`                                     | `User` (ID, RoleID, RoleCode, Email, PasswordHash, IsActive), `LoginInput`, `TokenPair`, `AccessClaims` (UserID, Role), `RefreshToken` (UserID, ExpiresAt)                                                                                                                 |
| `internal/repository/repository.go`                           | добавить `UserRepository` (`GetByEmail`, `GetByID`; роль подтягивается JOIN с `roles`) и `RefreshTokenRepository` (`Create`, `Consume` (получить и удалить за одну команду), `Delete`, `RevokeAllForUser`) |
| `internal/repository/user/`, `internal/repository/refresh/` | `user` на pgx (`repository.go`, `db_model.go`), `refresh` на Redis (`repository.go`, клиент `*redis.Client`)                                                                                                                                                   |
| `internal/service/service.go`                                 | интерфейс `AuthService` (`Login`, `Refresh`, `Logout`, `Authenticate`)                                                                                                                                                                                            |
| `internal/service/auth/`                                      | `service.go`, `login.go`, `refresh.go`, `logout.go`, `tokens.go`, `authenticate.go`                                                                                                                                                                                      |
| `internal/handler/response/response.go`                       | `JSON`, `Error`, `ServiceError`                                                                                                                                                                                                                                                |
| `internal/handler/v1/`                                        | `handler.go`, `request.go`, `dto.go`, `auth.go`                                                                                                                                                                                                                              |
| `internal/handler/middleware/`                                | `auth.go` (`Auth`, `RequireRole`, `ClaimsFromContext`), `logging.go`, `recover.go`, `cors.go`                                                                                                                                                                          |
| `internal/handler/handler.go`, `internal/app/app.go`        | подключить новые репозитории, сервис и маршруты                                                                                                                                                                                             |

**Правила:**

- `POST /api/v1/auth/login`, тело `{"email":"hr@company.local","password":"<пароль из go run ./cmd/seed>"}`, ответ 200 `{"access_token":"...","refresh_token":"..."}`.
- Email приводится к нижнему регистру. Неизвестный email, неверный пароль и `is_active = false` дают одну и ту же ошибку 401 `invalid email or password`.
- **Access-токен** — JWT HS256 на 15 минут. Claims: `sub` (id пользователя), `role` (`hr` или `employee`), `type=access`, `iat`, `exp`.
- **Refresh-токен** — случайная строка (32 байта из `crypto/rand`, base64url), не JWT. В Redis хранится ключ `refresh:<SHA-256 хэш>` со значением id пользователя и TTL 30 дней (сам токен не хранится). Хэш также добавляется в множество `refresh_user:<id>`.
- `POST /auth/refresh`, тело `{"refresh_token":"..."}`: найти по хэшу и удалить одной атомарной командой (`GETDEL`): истёк по TTL или уже использован, значит ключа нет, ошибка; иначе выдать новую пару. Повторное использование старого → 401.
- `POST /auth/logout`, тело `{"refresh_token":"..."}` → удалить ключ (`DEL`), ответ 204 (даже если ключа уже нет).
- `Auth` читает `Authorization: Bearer <access>`, без токена или с плохим → 401. `RequireRole("hr")` для другой роли → 403.
- Токен не логировать, пароль не возвращать в ответах.

**Проверка в Postman:**

1. login с верным паролем → **200**; с неверным → **401**; без поля `password` → **400**.
2. refresh с полученным refresh-токеном → **200** (новая пара); тот же старый ещё раз → **401**.
3. logout → **204**; refresh этим же токеном после logout → **401**.
4. Проверка `Auth` и `RequireRole` — на шаге 4 (отделы).

**Готово, когда:** все пункты проверки дают ожидаемые коды, после login в Redis есть ключ (`docker exec hr-redis redis-cli --scan --pattern 'refresh:*'`), после logout и после refresh старого ключа нет.

### Шаг 4. Отделы (CRUD, только HR)

**Цель:** первый полный CRUD. Этот шаблон повторяется в шагах 5, 6, 10, 11.

**Файлы:**

| Файл                                          | Что написать                                                                                                                                                                    |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/models/department.go`                 | `Department` (ID, Name, CreatedAt, UpdatedAt), `DepartmentInput` (Name)                                                                                                                |
| `internal/repository/repository.go`             | `DepartmentRepository`: `Create`, `List`, `GetByID`, `Update`, `Delete`                                                                                                        |
| `internal/repository/department/`               | SQL; код `23505` → `ErrConflict`; на удалении код `23503` → `ErrConflict`; `Delete` без строк → `ErrNotFound`                                       |
| `internal/service/service.go`                   | `DepartmentService`                                                                                                                                                                      |
| `internal/service/department/`                  | `service.go`, `create.go`, `get.go`, `list.go`, `update.go`, `delete.go`; валидация: trim, длина имени 1–100 символов (`utf8.RuneCountInString`) |
| `internal/handler/v1/department.go`, `dto.go` | 5 обработчиков и структуры запроса и ответа                                                                                                            |
| `internal/handler/handler.go`                   | 5 маршрутов за `requireAuth` + `requireHR`                                                                                                                                  |

**Эндпоинты:** `POST /api/v1/departments` (201), `GET /api/v1/departments` (200, список, пустой `[]`), `GET /api/v1/departments/{id}` (200), `PUT /api/v1/departments/{id}` (200), `DELETE /api/v1/departments/{id}` (204). Тело: `{"name":"IT"}`. Ответ: `{"id":1,"name":"IT","created_at":"...","updated_at":"..."}`.

**Проверка в Postman** (токен HR в переменной `token`, Bearer `{{token}}`):

1. POST `{"name":"IT"}` → **201**; то же имя ещё раз → **409**; пустое имя → **400**.
2. GET списка → **200**; GET `/999` → **404**; GET `/abc` → **400**.
3. PUT `/1` с новым именем → **200**.
4. DELETE `/1` → **204**, повторный → **404**.
5. Без токена → **401**. (403 для роли `employee` проверяем после шага 6.)
6. Удаление отдела со сотрудниками → **409** (проверяем после шага 6).

**Готово, когда:** пункты 1–5 дают ожидаемые коды.

### Шаг 5. Должности (CRUD, только HR)

**Цель:** повторить шаблон шага 4 самостоятельно, без подсказок.

**Файлы:** `models/position.go`, `PositionRepository` в `repository.go`, `repository/position/`, `PositionService` в `service.go`, `service/position/` (те же файлы, что у отделов), `handler/v1/position.go`, 5 маршрутов `/api/v1/positions`.

**Тело:** `{"name":"Backend Developer","base_salary":5000}`. Валидация: имя 1–100 символов, `base_salary` ≥ 0. Деньги в учебном проекте храним в Go как `float64`, в SQL читаем `base_salary::float8`.

**Проверка:** те же коды, что у отделов, плюс отрицательный `base_salary` → **400**.

**Готово, когда:** все коды совпали.

### Шаг 6. Сотрудники и профиль /me

**Цель:** HR создаёт сотрудника с аккаунтом; сотрудник входит и видит только себя.

**Файлы:**

| Файл                                                   | Что написать                                                                                                                                                                                                                    |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/models/employee.go`                            | `Employee` (ID, UserID, Email, FirstName, LastName, Phone, DepartmentID, DepartmentName, PositionID, PositionName, HireDate, TerminationDate, Status), `EmployeeInput` для создания (+ Email, Password), `EmployeeUpdate` |
| `internal/repository/repository.go`                      | `EmployeeRepository`: `Create` (одна транзакция: INSERT в `users`, потом в `employees`), `GetByID`, `GetByUserID`, `List`, `Update`, `Terminate`                                                    |
| `internal/repository/employee/`                          | SQL с JOIN на `users`, `departments`, `positions`; `pgx.Tx` (`db.Begin`) в `Create` и `Terminate`                                                                                                                       |
| `internal/service/employee/`                             | `service.go`, `create.go`, `get.go`, `list.go`, `update.go`, `delete.go`, `me.go`                                                                                                                                            |
| `internal/handler/v1/employee.go`, `me.go`, `dto.go` | обработчики и DTO                                                                                                                                                                                                              |
| `internal/handler/handler.go`                            | маршруты                                                                                                                                                                                                                           |

**Эндпоинты:**

- `POST /api/v1/employees` (HR) → 201. Тело: `{"email":"ivan@company.local","password":"Pass12345","first_name":"Ivan","last_name":"Petrov","phone":"+992900000000","department_id":1,"position_id":1,"hire_date":"2026-01-15"}`. Создаётся `users` (роль `employee`) и `employees` в одной транзакции.
- `GET /api/v1/employees/{id}` (HR) → 200.
- `PUT /api/v1/employees/{id}` (HR) → 200. Тело: `department_id`, `position_id`, `first_name`, `last_name`, `phone` (обязателен), `status`.
- `DELETE /api/v1/employees/{id}` (HR) → 204. Не удаляет строку: `status = terminated`, `termination_date` = сегодня, `users.is_active = false`, отозвать все refresh-токены пользователя (удалить ключи из множества `refresh_user:<id>`).
- `GET /api/v1/me` (любая роль) → 200 `{"id":2,"email":"...","role":"employee","employee":{...}}`; у HR без записи в `employees` поле `employee` = `null`.

**Валидация:** email корректный и уникальный (409), пароль 8–72 байта, имена 1–50 символов, `phone` обязателен, 1–20 символов после trim (при создании и при обновлении; пустой даёт 400), `hire_date` формата `YYYY-MM-DD`, отдел и должность существуют (иначе 400), `status` из списка.

**Проверка в Postman:**

1. HR: POST `/employees` → **201**; повтор с тем же email → **409**; с несуществующим `department_id` → **400**.
2. Login как созданный сотрудник → **200**; `GET /me` → **200** с его данными.
3. Токеном сотрудника: `GET /employees`, `POST /departments`, `DELETE /employees/1` → все **403**.
4. HR: PUT → **200**; DELETE → **204**; login уволенного → **401**.
5. Удаление отдела с этим сотрудником → **409**.

**Готово, когда:** все пункты проверки верны. Это ядро проекта (весь этап 1 кроме поиска).

### Шаг 7. Поиск, фильтры и пагинация

**Цель:** `GET /api/v1/employees` принимает параметры.

**Параметры:** `q` (подстрока в имени, фамилии или email, `ILIKE`), `department_id`, `position_id`, `status`, `page` (по умолчанию 1), `limit` (по умолчанию 20, максимум 100). Сортировка: `last_name`, потом `first_name`.

**Ответ:** `{"items":[...],"total":42,"page":1,"limit":20}`.

**Файлы:** `models/employee.go` (`EmployeeFilter`, `EmployeePage`), `repository/employee/` (динамический WHERE с параметрами `$1, $2...`, данные не вставлять в строку SQL), `service/employee/list.go` (ограничения page/limit), `handler/v1/employee.go` (разбор `c.Query(...)` и `c.DefaultQuery(...)`).

**Проверка:** `?q=iva` находит Ивана; `?department_id=1&status=active`; `?limit=2&page=2`; `?limit=1000` → **400**; `?page=0` → **400**.

**Готово, когда:** все комбинации дают ожидаемый `total` и список.

### Шаг 8. Документация API

**Цель:** описать готовое API и посмотреть в Swagger UI.

**Файлы:** `docs/openapi.yaml` (OpenAPI 3.0.3: `servers`, `tags`, все `paths`, `components/schemas` для DTO, `components/securitySchemes` — `bearerAuth`, ответы 400/401/403/404/409), `swagger-ui/docker-compose.yaml` (образ `swaggerapi/swagger-ui`, порт 8081). В `cors.go` разрешить origin `http://localhost:8081`.

**Проверка:** `docker compose -f swagger-ui/docker-compose.yaml up -d`, открыть `http://localhost:8081`, в Swagger UI нажать **Authorize**, вставить access-токен и выполнить `GET /employees`. Импортировать файл в Postman в формате JSON (YAML в расширении VS Code не импортируется).

**Готово, когда:** все эндпоинты этапа 1 видны в Swagger UI и выполняются из него.

### Шаг 9. Unit-тесты

**Цель:** проверять бизнес-логику без БД.

**Файлы:** `internal/service/auth/login_test.go`, `refresh_test.go`, `internal/service/employee/create_test.go`. В каждом пакете моки репозиториев (структуры, реализующие интерфейсы на map в памяти), табличные тесты (`[]struct{name string; ...}` + `t.Run`).

**Случаи:** login (успех, неверный пароль, неактивный пользователь); refresh (успех, повторное использование, истёкший токен); создание сотрудника (некорректный email, короткий пароль, пустое имя).

**Проверка:** `go test ./...`.

**Готово, когда:** все тесты зелёные.

### Шаг 10. Этап 2: зарплаты и отчёт по отделам

**Миграция **`migrations/004_salaries.sql`**:**

```
CREATE TABLE salaries (
    id             SERIAL PRIMARY KEY,
    employee_id    INT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    effective_date DATE NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (employee_id, effective_date)
);
CREATE INDEX idx_salaries_employee_id ON salaries(employee_id);
```

**Файлы:** `models/salary.go`, `SalaryRepository`, `repository/salary/`, `SalaryService` и `ReportService`, `service/salary/`, `service/report/`, `handler/v1/salary.go`, `handler/v1/report.go`.

**Эндпоинты:**

- `POST /api/v1/employees/{id}/salaries` (HR), тело `{"amount":5000,"effective_date":"2026-02-01"}` → 201; та же дата ещё раз → 409; `amount <= 0` → 400.
- `GET /api/v1/employees/{id}/salaries` (HR) → 200, новые первыми.
- `GET /api/v1/me/salaries` (сотрудник, свои) → 200.
- `GET /api/v1/reports/departments` (HR) → 200 `[{"department_id":1,"department":"IT","employees_count":3,"total_salary":15000,"avg_salary":5000}]`.

**SQL отчёта** (текущая зарплата = последняя по дате не позже сегодня, уволенные не учитываются):

```
SELECT d.id, d.name,
       COUNT(e.id) AS employees_count,
       COALESCE(SUM(cs.amount), 0)::float8 AS total_salary,
       COALESCE(AVG(cs.amount), 0)::float8 AS avg_salary
FROM departments d
LEFT JOIN employees e ON e.department_id = d.id AND e.status <> 'terminated'
LEFT JOIN LATERAL (
    SELECT s.amount FROM salaries s
    WHERE s.employee_id = e.id AND s.effective_date <= CURRENT_DATE
    ORDER BY s.effective_date DESC LIMIT 1
) cs ON true
GROUP BY d.id, d.name
ORDER BY d.name;
```

**Проверка:** создать 2 отдела и 3 сотрудников с зарплатами, отчёт совпадает с ручным подсчётом; сотрудник на чужую зарплату → **403**.

**Готово, когда:** коды и цифры отчёта верны.

### Шаг 11. Этап 2: отпуска и больничные

**Миграция **`migrations/005_time_off.sql`**:**

```
CREATE TABLE time_off (
    id          SERIAL PRIMARY KEY,
    employee_id INT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    approved_by INT REFERENCES users(id),
    leave_type  VARCHAR(20) NOT NULL CHECK (leave_type IN ('vacation', 'sick')),
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT time_off_dates CHECK (end_date >= start_date)
);
CREATE INDEX idx_time_off_employee_id ON time_off(employee_id);
CREATE INDEX idx_time_off_status ON time_off(status);
```

**Файлы:** `models/timeoff.go`, `TimeOffRepository`, `repository/timeoff/`, `TimeOffService`, `service/timeoff/`, `handler/v1/timeoff.go`.

**Эндпоинты и правила:**

- `POST /api/v1/time-off` (сотрудник), тело `{"leave_type":"vacation","start_date":"2026-07-01","end_date":"2026-07-14"}` → 201, `status = pending`. `employee_id` берётся из токена, не из тела. `end_date < start_date` → 400. Пересечение с своей заявкой `pending` или `approved` → 409.
- `GET /api/v1/time-off` — HR видит все (фильтры `status`, `employee_id`), сотрудник только свои → 200.
- `PATCH /api/v1/time-off/{id}/status` (HR), тело `{"status":"approved"}` или `"rejected"` → 200. Менять можно только `pending` (иначе 409). В `approved_by` пишется id HR из токена.

**Проверка:** сотрудник создаёт заявку → 201; HR одобряет → 200; повторное изменение → 409; сотрудник вызывает PATCH → 403; чужие заявки сотруднику не видны.

**Готово, когда:** все коды верны.

### Шаг 12. Этап 3: история перемещений и CSV

**Миграция **`migrations/006_position_history.sql`**:**

```
CREATE TABLE employee_position_history (
    id            SERIAL PRIMARY KEY,
    employee_id   INT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    department_id INT NOT NULL REFERENCES departments(id),
    position_id   INT NOT NULL REFERENCES positions(id),
    start_date    DATE NOT NULL,
    end_date      DATE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT history_dates CHECK (end_date IS NULL OR end_date >= start_date)
);
CREATE INDEX idx_history_employee_id ON employee_position_history(employee_id);
```

**История:** при создании сотрудника добавить строку (`start_date = hire_date`, `end_date = NULL`). При смене отдела или должности в `PUT /employees/{id}`, в одной транзакции: закрыть текущую строку (`end_date = сегодня`) и добавить новую. `GET /api/v1/employees/{id}/history` (HR) → 200, новые первыми.

**Экспорт:** `GET /api/v1/employees/export.csv` (HR), фильтры такие же, как у списка. Заголовки `Content-Type: text/csv; charset=utf-8`, `Content-Disposition: attachment; filename="employees.csv"`. Колонки: `id,email,first_name,last_name,phone,department,position,hire_date,status`. Пишем через `encoding/csv`.

**Импорт:** `POST /api/v1/employees/import` (HR), `multipart/form-data`, поле `file`, максимум 1 МБ и 1000 строк. Колонки: `email,password,first_name,last_name,phone,department,position,hire_date` (отдел и должность по имени, они должны существовать; пустой `phone` в строке это ошибка строки). Каждая строка в своей транзакции, ошибка строки не останавливает остальные. Ответ 200: `{"created":2,"errors":[{"row":3,"error":"email already registered"}]}`.

**Файлы:** `models/history.go`, `repository/history/`, `service/employee/history.go`, `service/employee/export.go`, `service/employee/import.go`, `handler/v1/history.go`, `handler/v1/employee_csv.go`.

**Проверка:** сменить сотруднику должность → в истории 2 строки, первая с `end_date`; экспорт скачивается и открывается в Excel; импорт файла из 3 строк с одной ошибкой даёт `created: 2` и одну ошибку.

**Готово, когда:** все проверки дают ожидаемые коды. Проект закончен: обновить `README.md` и `docs/openapi.yaml`.

## Правила работы и с чего начать

- Один шаг за раз. Дальше не идём, пока проверка в Postman не прошла.
- Код всегда полный и запускаемый, ответы короткие, без воды.
- Стиль кода как в demo-project: интерфейсы в одном файле пакета, `var _ Interface = (*Service)(nil)`, ошибки через `fmt.Errorf("...: %w", err)`.
- Windows + Git Bash. Порты 5432 и 6379 заняты локально, поэтому Redis в Docker на 6380. `make` может отсутствовать: дублировать команды `docker compose`.
- Postman в VS Code: скрипты переменные не пишут. Токен вставляем в переменную `token` вручную, в защищённых запросах Bearer `{{token}}`.

**Первый шаг сегодня:** создать папку `hr-system`, выполнить `go mod init hr-system`, запросить у Claude шаг 1 плана (каркас) с полным кодом.

## Принятые решения

Уточнения к ТЗ, чтобы везде было одно и то же:

- Телефон сотрудника обязателен: `employees.phone VARCHAR(20) NOT NULL`, в Go `string` (не указатель), валидация 1–20 символов после trim. Решение автора проекта, ментору показать.
- Дубликат (email, имя отдела и т.п.) и связанные записи при удалении: `models.ErrConflict` → HTTP 409. Отдельного `ErrEmailTaken` нет.
- Тип id в Go: `int64` (как в demo-project2), в Postgres остаётся `SERIAL`.
- Ошибки запросов логируются в одном месте, `handler/response/response.go`: 4xx как Warn, 500 как Error.
- Тексты ошибок короткие, на английском, в формате `{"error":"текст"}` (например `invalid email or password`).
- `REDIS_PASSWORD` локально пустой.
- `requireHR` — локальная переменная в `InitRoutes()`, собранная из `requireAuth` и `RequireRole("hr")`.

## Дополнения (после шага 9)

Дополнения применяются только после шага 9. До этого работаем строго по шагам выше. Шаги, таблицы и порядок выше не менялись. Где пункт уточняет прежний шаг, это сказано в самом пункте; после шага 9 при расхождении действует дополнение. Кода здесь нет: код пишу сам.

### 1. docs/architecture.md

Короткий документ «как устроено и почему» по образцу `architecture.md` из примера. Создаём после шага 9, чтобы описать то, что уже работает.

- **Разделы:** 1) Слои: схема handler → service → repository → models, таблица «делает / не делает», дерево папок, где лежат интерфейсы (`service.go`, `repository.go`) и сборка (`app.go`); 2) Как написан метод сервиса: `ctx`, `CurrentUser`, `Validate()`, ошибки с `%w`; 3) Бизнес-правила: роли и права, увольнение вместо удаления; 4) Потоки: login, refresh, logout; 5) Решения; 6) Компромиссы.
- **Потоки:** login: `Validate` → поиск по email → bcrypt → access и refresh → хэш refresh в Redis (неизвестный email, неверный пароль и неактивный аккаунт дают одну и ту же 401). Refresh: токен из тела → `GETDEL` → пользователь → новая пара. Logout: `DEL`, ответ 204 даже без ключа.
- **Решения** (таблица «решение — почему»): refresh в Redis (решение ментора), хэш вместо токена, одноразовый refresh через `GETDEL`, увольнение вместо удаления, `ErrConflict` для дублей и связанных записей, `int64` для id, лог ошибок в одном месте (`response.go`).
- **Redis, ключи:** `refresh:<SHA-256 хэш>` → id пользователя, TTL 30 дней (`JWT_REFRESH_TTL`); `refresh_user:<id>` — множество хэшей, чтобы при увольнении отозвать все токены сотрудника.
- **Redis, что и почему:** хранится хэш, а не токен: утечка Redis не даёт рабочих токенов. Refresh — случайная строка, не JWT. Таблицы `refresh_tokens` нет, истёкшие токены удаляет TTL.
- **Redis, одноразовость:** `GETDEL` забирает и удаляет ключ одной командой: из двух одновременных refresh с одним токеном пройдёт только первый; повторно предъявленный старый токен → 401.
- **Компромиссы:** access — обычный JWT, в Redis его нет, поэтому после logout и увольнения он действует до конца своих 15 минут; Redis недоступен — login, refresh и logout не работают; миграции применяются только при первом создании базы (для production нужен migration-tool, например goose или atlas); деньги в `float64`, а seed-HR с известным паролем — только для локальной разработки.

### 2. `models.CurrentUser` и `Validate()`

**`CurrentUser`** — «кто сейчас делает запрос».

- Структура в `internal/models/user.go`: `ID` (`int64`, из `sub` access-токена) и `Role` (код роли: `hr` или `employee`). Метод `IsHR()` возвращает true для роли `hr`. Больше в ней ничего нет: ни email, ни данных из БД.
- Handler после `Auth` собирает её из `ClaimsFromContext` (общий помощник в `request.go`) и передаёт в сервис вторым параметром после `ctx`. Так сервис ничего не знает про JWT и HTTP.
- Нужна там, где маршрут открыт обеим ролям и результат зависит от роли: `GET /me`, зарплаты и отпуска (шаги 10–11: HR видит всё, сотрудник только своё). Маршруты только для HR по-прежнему закрывает `requireHR` (403), туда `CurrentUser` можно не передавать.
- Новые права добавляем методами `CurrentUser` рядом с `IsHR()`, а не сравнениями `Role == "hr"` по сервисам.

**`Validate()`** — проверка входных данных.

- Метод у входных структур в `models` (`LoginInput`, `DepartmentInput`, `PositionInput`, `EmployeeInput`, `EmployeeUpdate`). Сначала нормализация (trim, email в нижний регистр; пароль не trim-им), потом проверка (длины, диапазоны, допустимый `status`). Получатель — указатель, чтобы нормализованные значения остались в структуре. Ошибка строится на `ErrInvalidInput`, текст короткий, на английском.
- Service: `in.Validate()` — первая строка метода, дальше правила, которым нужна БД (отдел существует, email свободен).
- Handler `Validate()` не вызывает: разбирает JSON, `{id}`, query и даты, копирует DTO в Input. Ошибка формата → 400 сразу; `ErrInvalidInput` из сервиса → 400 через `ServiceError`.
- Это уточнение шагов 4–6: trim и проверки длин, описанные там в service, теперь живут в `Validate()`. Коды ответов те же.

### Правила `Validate()` по структурам

Метод у входной структуры вызывается первой строкой сервиса, нормализует поля (trim, нижний регистр) и возвращает `fmt.Errorf("%w: описание", models.ErrInvalidInput)`. Описание на английском: оно уходит клиенту как есть. Длины считаем в символах (`utf8.RuneCountInString`), кроме пароля: он считается в байтах, потому что bcrypt принимает не больше 72 байт.

| Структура | Правила |
|---|---|
| `LoginInput` | email: trim и нижний регистр, не пустой; пароль не пустой и **не обрезается** |
| `DepartmentInput` | `name`: trim, 1–100 символов |
| `PositionInput` | `name`: trim, 1–100 символов; `base_salary >= 0` |
| `EmployeeInput` | email: trim, нижний регистр, корректный адрес (`net/mail`), в домене есть точка, до 100 символов; пароль 8–72 байта; `first_name`, `last_name`: trim, 1–50; `phone`: trim, 1–20; `department_id > 0`, `position_id > 0`; `hire_date` задана |
| `EmployeeUpdate` | `first_name`, `last_name`, `phone` и id как выше; `status` из `active`, `on_leave`, `terminated` |
| `SalaryInput` | `amount > 0`; `effective_date` задана |
| `TimeOffInput` | `leave_type` из `vacation`, `sick`; обе даты заданы; `end_date` не раньше `start_date` |
| `TimeOffFilter` | `status` пустой или `pending`, `approved`, `rejected` |
| решение по заявке | `approved` или `rejected` (`models.ValidateTimeOffDecision`) |

Существование отдела и должности, запрет `terminated` через `PUT` и уникальность email проверяет сервис и база, не `Validate()`.

### 3. Тесты шага 9 (уточнение)

- **Правила:** табличные тесты (`[]struct{name ...}` + `t.Run`), внешний пакет (`auth_test`, `employee_test`, `models_test`), сервис создаётся через `NewService` с моками. Ошибки сравниваем через `require.ErrorIs`. Хэш пароля в тесте считаем `bcrypt` с `MinCost`, чтобы тест не тормозил.
- **Моки:** вместо ручных моков на map (так было в шаге 9) используем mockery: он генерирует моки интерфейсов из `repository/repository.go` в пакет `mocks` (`internal/repository/mocks`), ожидания пишем через `EXPECT()`. В Makefile цели `mocks` и `test` (`go test ./...`). После любого изменения интерфейса: `make mocks` и коммит сгенерированных файлов. Сгенерированные моки зависят от `testify`, он появится в `go.mod`.
- **auth, login:** успех (есть access и refresh, refresh сохранён один раз); email в верхнем регистре ищется в нижнем; неизвестный email, неверный пароль и `is_active = false` дают одну и ту же `ErrInvalidCredentials`; пустой email или пароль — `ErrInvalidInput`; сбой репозитория (не `ErrNotFound`) возвращается ошибкой, а не `ErrInvalidCredentials`.
- **auth, refresh / logout / authenticate:** refresh: успех (выдана новая пара, новый refresh сохранён); `Consume` вернул `ErrNotFound` → `ErrInvalidToken` (кейсы «истёкший» и «повторное использование»: в Redis ключа нет в обоих, мок один и тот же; для повторного использования мок отвечает успехом на первый вызов и `ErrNotFound` на второй); сбой `GetByID` после `Consume` — пара не выдаётся. Logout: успех и сбой Redis. Authenticate (без моков): верный access → id и роль; чужая подпись, истёкший, `type` не `access` → `ErrInvalidToken`.
- **employee, create:** успех (в `Create` уходят email в нижнем регистре и bcrypt-хэш, а не пароль); `ErrInvalidInput` без обращения к репозиториям: некорректный email, пароль 7 байт, пароль 73 байта, пустое имя, имя из одних пробелов; отдел или должность не существуют → `ErrInvalidInput`; `Create` вернул `ErrConflict` (email занят) → `ErrConflict`; другая ошибка репозитория возвращается обёрнутой.
- **employee, delete и me:** delete (увольнение): сотрудник не найден → `ErrNotFound`; успех → вызваны `Terminate` и `RevokeAllForUser`. Me: у HR без записи в `employees` это не ошибка, `employee = nil`.
- **models:** тесты `Validate()` лежат в `internal/models/*_test.go`. Границы проверяем здесь, а в сервисных тестах берём по одному кейсу на ошибку. Длины на границах (0, 1, максимум, максимум+1; отдел и должность — 100, имя и фамилия — 50, `phone` — 1–20, пустой отклоняется; кириллица считается символами), пароль 7 / 8 / 72 / 73 байта (кириллица — по 2 байта), `base_salary` −1 и 0 (0 допустим), `status` вне списка, email с пробелами и заглавными приводится к нижнему регистру. `CurrentUser.IsHR`: `hr` → true, `employee` и пустая роль → false.
- **Проверка:** `go test ./...` зелёный. В кейсах с ошибкой валидации у мока нет `EXPECT()`: лишний вызов репозитория уронит тест.

### 4. Middleware RequestID

- **Что делает:** на каждый запрос создаёт случайный id (`crypto/rand`, в hex), кладёт его в `context` и отдаёт в заголовке ответа `X-Request-ID`. Входящий `X-Request-ID` не принимаем: id всегда свой. Файл: `internal/handler/middleware/request_id.go`.
- **Где подключается:** в `InitRoutes()` первым в общей цепочке: RequestID → CORS → Logging → Recover. Тогда id есть и у `Logging`, и у `Recover`, и у `response.go`.
- **Как попадает в лог:** ключ контекста и функции `WithRequestID` / `RequestIDFromContext` лежат в `pkg/logger` (так `pkg` не импортирует `internal`). Логгер сам добавляет поле `request_id` в каждую запись, написанную с `ctx`.
- **Правило для кода:** в обработке запроса логируем только с контекстом: `slog.WarnContext(c.Request.Context(), ...)`. Это касается `Logging`, `Recover` и `response.go` (`Error` и `ServiceError` получают `*gin.Context`).
- **Результат:** строка запроса от `Logging` и строка ошибки от `response.go` имеют один `request_id`: по id из ответа в Postman находим причину в логе. Строки без запроса (`server started`, `shutting down`) id не имеют.
- **Проверка:** любой запрос → в ответе есть `X-Request-ID`. `GET /api/v1/departments/999` с токеном HR → 404, и `grep <id> app.log` (при `APP_ENV=local`) показывает две строки с этим id: запрос и ошибку.

### 5. Чек-лист перед коммитом и «чего не делать»

По образцу `feature_rules.md`, под слои и правила этого проекта.

**Чек-лист:**

- [ ] **handler:** только разбор запроса (JSON, `{id}`, query, даты) и ответ; нет бизнес-проверок, SQL и своего `slog`. **service:** `in.Validate()` первой строкой, права и состояние проверены, ошибки обёрнуты `%w`, нет HTTP, JSON, SQL и логов.
- [ ] **models:** правила и нормализация в `Validate()`, без ввода-вывода и тегов `json`/`db`. **repository:** простой SQL; `ErrNoRows` → `ErrNotFound`, `23505` и `23503` → `ErrConflict`; после смены интерфейса `make mocks`.
- [ ] Дубликат и связанные записи — только `ErrConflict` (409). Id — `int64` во всех слоях и в DTO. Маршрут добавлен в `handler.go` с нужной обёрткой (`requireAuth`, `requireHR`).
- [ ] `docs/openapi.yaml` обновлён, `go vet ./...` и `go test ./...` зелёные, ручка проверена в Postman.

**Чего не делать:**

- Бизнес-условия в `WHERE` и «сначала `SELECT`, потом `INSERT`» в repository: уникальность и связи проверяет БД, а repository лишь переводит её ошибки в `ErrConflict`. SQL вне repository.
- Проверять в handler роль (её режет `requireHR` в роутере) и владельца (это service). Свой `slog` в handler, service и repository: ошибки запроса логирует только `response.go` (4xx — Warn, 500 — Error), свои строки пишут ещё `Logging` и `Recover`; запуск и остановка в `app.go` и `main.go` — исключение.
- Отдельная ошибка под частный случай (`ErrEmailTaken`), `int` или `int32` для id, значения в SQL строкой вместо `$1, $2...`, пароли, хэши паролей и токены в логах и ответах.
- Ручные моки вместо mockery, ORM, generic-репозитории, библиотеки валидации, DI-контейнер и обёртки над bcrypt и JWT «на будущее».

### 6. docs/db-diagram.md

- Один файл, одна диаграмма: блок кода `mermaid` с `erDiagram`. Redis не рисуем.
- **Таблицы этапа 1** (миграции 001–003): `roles`, `users`, `departments`, `positions`, `employees`. В каждой все поля из ТЗ с типом и пометками PK / FK / UK. Тип `NUMERIC(12,2)` пишем как `NUMERIC`: запятая в типе может ломать разбор mermaid.
- **Связи** (из таблицы «Связи» в ТЗ): `roles ||--o{ users`, `users ||--o| employees` (один к одному: у HR может не быть записи в `employees`), `departments ||--o{ employees`, `positions ||--o{ employees`. Подпись связи — имя FK-колонки: `role_id`, `user_id`, `department_id`, `position_id`.
- **Не рисуем:** Redis и refresh-токены (это не таблицы), а также `salaries`, `time_off`, `employee_position_history`: они относятся к этапам 2–3 и добавятся на диаграмму вместе с миграциями шагов 10–12.
- Под диаграммой одна строка текста: refresh-токены лежат в Redis, см. `architecture.md`.
- **Проверка:** поля и типы совпадают с `migrations/001–003`; диаграмма открывается в превью (VS Code с расширением для Mermaid или mermaid.live).

## Статус реализации и отклонения (на 2026-11-03)

Раздел описывает, что уже написано в коде и чем реализация отличается от текста выше. Если текст выше и этот раздел расходятся, верен этот раздел.

### Что готово

- **Код шагов 1–12** и пунктов «Дополнений (после шага 9)»: `architecture.md`, `CurrentUser` и `Validate()`, тесты на mockery, `RequestID`, `db-diagram.md`.
- **Тесты:** сервисы (`auth`, `department`, `position`, `employee` с импортом и экспортом CSV, `salary`, `report`, `timeoff`), HTTP-слой на настоящем Gin (`internal/handler/handler_test.go`: маршруты, авторизация, коды ответов, CORS, паника, лимит тела, импорт), `models` (`Validate()`), `dberr`, `pkg/logger`.
- **Документация:** `docs/openapi.yaml` описывает все эндпоинты, `docs/architecture.md`, `docs/db-diagram.md`, `README.md`.

### Что проверено, а что нет

Проверено в изолированной среде (Go 1.22, Gin 1.9.1, mockery 2.43.2): `go vet ./...`, `go test ./...` (все пакеты зелёные), сборка всех пакетов, регистрация маршрутов на настоящем Gin (в том числе `/employees/export.csv` рядом с `/employees/:id`).

**Не проверено** (нужны твоя машина, Docker и Postman): запуск с настоящими Postgres и Redis (то есть SQL-запросы репозиториев), сборка с Go 1.26 и Gin 1.12, проверки «Проверка:» шагов 1, 3–7 и 10–12. Шаг считается закрытым только после них.

### Отклонения от текста ТЗ

1. **Gin вместо `net/http`** (решение ментора): раздел «HTTP-слой на Gin» выше заменил отдельный файл `TZ-gin-addendum.md`.
2. **`Validate()` и `CurrentUser` из «Дополнений»** применены сразу, а не после шага 9. Проверка длин и форматов живёт в `models`. Правило в `copilot-instructions.md` «проверка длин только в service» нужно считать устаревшим.
3. **Env-файл вне проекта:** `config.Load()` читает файл по переменной `ENV_FILE`, запасной путь `.env`. Если `ENV_FILE` задан, а файла нет, это ошибка запуска.
4. **Первого HR создаёт `cmd/seed`**, а не INSERT в миграции: пароль вводится с клавиатуры, в БД идёт только bcrypt-хэш, `ON CONFLICT (email) DO NOTHING`, email по умолчанию `hr@company.local` (флаг `-email`). Пароль из старого текста ТЗ больше не используется.
5. **`PUT /employees/{id}`:** `status = terminated` даёт 400, увольняет только `DELETE`. У уволенного данные менять нельзя (409). Повторный `DELETE` по уволенному даёт 409.
6. **`JWT_SECRET` короче 32 байт:** приложение не запускается.
7. **`/health`:** при недоступном Postgres или Redis отвечает 503 `{"error":"postgres is unavailable"}` (или `redis`).
8. **`request_id`:** `RequestID` кладёт id в контекст запроса (`logger.WithRequestID`) и в заголовок `X-Request-ID`; обработчик логгера добавляет `request_id` в каждую строку, записанную с этим контекстом. Входящий `X-Request-ID` не принимается.
9. **Язык сообщений:** все тексты в коде (ошибки `fmt.Errorf("...: %w")`, строки лога, вывод `cmd/seed` и `cmd/api`) на английском и короткие, в виде «глагол объект»: `create department: %w`, `shutdown server: %w`, `server started`, `shutting down`. Тексты, которые видит клиент API (тела ответов, ошибки `Validate()`, `errors[].error` при импорте), тоже английские. Комментарии в коде не пишем.
10. **Имена полей сотрудника в JSON:** `department_name` и `position_name` (как в `openapi.yaml`).
11. **Моки:** mockery v2.43.2, настройки в `.mockery.yaml`, сгенерированный код лежит в репозитории (`internal/repository/mocks`, `internal/service/mocks`). Перегенерация: `make mocks`.
12. **`refresh`:** пустой refresh-токен в теле даёт 400, неизвестный или использованный 401, токен уволенного 401. Старый токен одноразовый (`GETDEL`).
13. **История перемещений:** запись создаётся при приёме (`start_date = hire_date`); при смене отдела или должности в `PUT` текущая запись закрывается, открывается новая (в одной транзакции), дата смены `GREATEST(CURRENT_DATE, hire_date)`. Увольнение закрывает текущую запись той же датой, что `termination_date`.
14. **Отпуска:** `POST /time-off` от пользователя без записи сотрудника даёт 403; `approved_by` это id аккаунта HR; пересечение дат проверяется под `pg_advisory_xact_lock` по сотруднику (две параллельные заявки не пройдут обе); заявки со статусом `rejected` пересечение не блокируют.
15. **Импорт CSV:** `errors[].row` это номер строки в файле (заголовок это строка 1); допустимы разделители `,` и `;` и BOM (так сохраняет Excel); колонки в любом порядке; не больше 1000 строк и 1 МБ. Строки обрабатываются по очереди, каждая в своей транзакции. bcrypt занимает около 0,05–0,1 с на строку, поэтому для файла на 1000 строк нужно поднять `HTTP_WRITE_TIMEOUT` (по умолчанию 10 секунд).
16. **Экспорт CSV:** файл с BOM (чтобы Excel читал кириллицу); ячейки, начинающиеся с `=`, `+`, `-`, `@`, получают апостроф (защита от формул), телефоны вида `+992 90 000-00-00` не меняются.

### Новые файлы, которых нет в структуре выше

- `internal/repository/dberr/`, `internal/repository/{salary,timeoff,history}/`, `internal/repository/mocks/`, `internal/service/{salary,report,timeoff}/`, `internal/service/mocks/`.
- `internal/handler/v1/{dto,request,salary,report,timeoff,history,employee_csv}.go`, `internal/handler/middleware/{request_id,cors,logging,recover,auth}.go`.
- `internal/models/{timeoff,history}.go` и тесты `*_test.go` рядом с кодом.
- `.mockery.yaml`, `docs/architecture.md`, `docs/db-diagram.md`.

### Что осталось

1. Первый запуск и проверка в Postman (список ниже в README и в «Проверка:» каждого шага).
2. Вопросы ментору: `phone NOT NULL`; seed-программа вместо INSERT; свои `Logging` и `Recover` или готовые `gin.Logger()` и `gin.Recovery()`; `Validate()` в `models` без тегов `binding`; закрывать ли историю при увольнении (отклонение 13).
3. Коммит и push (файлы добавлять по именам).
