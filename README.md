# hr-system

REST API учёта сотрудников: отделы, должности, сотрудники, зарплаты, отпуска и больничные, история перемещений, импорт и экспорт CSV. Go, Gin, PostgreSQL, Redis, JWT.

Подробнее: `docs/TZ.md` (задание и принятые решения), `docs/architecture.md` (слои и правила), `docs/db-diagram.md` (схема БД), `docs/openapi.yaml` (API).

## Быстрый старт

1. **Env-файл вне проекта.** Скопируйте `.env.example` в папку вне репозитория (например `~/secrets/hr-system.env`), заполните значения и задайте путь:

   ```
   export ENV_FILE=~/secrets/hr-system.env
   export COMPOSE_ENV_FILES=~/secrets/hr-system.env
   ```

   `JWT_SECRET` не короче 32 символов (`openssl rand -hex 32`). Файл с настоящими значениями не коммитьте и никому не показывайте.
2. **Postgres и Redis** (миграции из `migrations/` применятся при первом создании базы):

   ```
   docker compose -f docker-compose.infra.yaml up -d --wait
   ```

   То же самое делает `make infra-up`. Пересоздать базу с нуля: `make infra-reset`.
3. **Зависимости:** `go mod tidy`.
4. **Первый HR** (пароль вводится с клавиатуры и нигде не сохраняется, в базу идёт только хэш):

   ```
   go run ./cmd/seed
   ```
5. **Запуск:** `go run ./cmd/api` (или `make run`). Проверка: `GET http://localhost:8080/health` должен вернуть `{"status":"ok"}`. Остановка по `Ctrl+C` пишет в лог `shutting down`.
6. **Swagger UI** (необязательно): `docker compose -f swagger-ui/docker-compose.yaml up -d`, затем `http://localhost:8081`, кнопка **Authorize**, вставить access-токен из `POST /api/v1/auth/login`.

## Проверки

```
make check      # сборка, go vet и тесты (то же: go build ./... && go vet ./... && go test ./...)
make mocks      # перегенерировать моки после изменения интерфейсов (нужен mockery v2.43.2)
```

Тесты не требуют БД. SQL и весь сценарий проверяются вручную в Postman по пунктам «Проверка:» из `docs/TZ.md`.

## API

Все маршруты под `/api/v1`, кроме `/health`. Ответы и ошибки в JSON, ошибка всегда `{"error":"текст"}`. Полное описание: `docs/openapi.yaml`.

| Группа                     | Маршруты                                                                                 | Кто                                                       |
| -------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------ |
| вход                         | `POST /auth/login`, `/auth/refresh`, `/auth/logout`                                        | все                                                       |
| профиль                   | `GET /me`, `GET /me/salaries`                                                                | любая роль                                          |
| отпуска                   | `POST /time-off`, `GET /time-off`                                                            | любая роль (сотрудник видит свои) |
| отпуска                   | `PATCH /time-off/{id}/status`                                                                  | HR                                                           |
| отделы, должности | `GET/POST /departments`, `GET/PUT/DELETE /departments/{id}`, то же для `/positions` | HR                                                           |
| сотрудники             | `GET/POST /employees`, `GET/PUT/DELETE /employees/{id}` (DELETE увольняет)          | HR                                                           |
| сотрудники             | `GET /employees/export.csv`, `POST /employees/import`, `GET /employees/{id}/history`       | HR                                                           |
| зарплаты и отчёт   | `GET/POST /employees/{id}/salaries`, `GET /reports/departments`                              | HR                                                           |

## Структура

```
cmd/api, cmd/seed        точки входа
internal/app             сборка зависимостей, запуск и остановка сервера
internal/config          настройки из env
internal/handler         маршруты, middleware, обработчики v1, формат ответов
internal/service         бизнес-логика
internal/repository      SQL (pgx) и Redis, моки в mocks/
internal/models          структуры, ошибки, Validate()
pkg                      logger, postgres, redis
migrations               SQL-миграции 001-006
docs                     ТЗ, архитектура, схема БД, openapi, шпаргалки
swagger-ui               compose для Swagger UI
```

## Безопасность

- Пароли хранятся только как bcrypt-хэш, refresh-токены только как SHA-256 в Redis.
- `.env`, пароли, хэши и токены не пишем в логи, ответы, README и чаты.
- Уволенный сотрудник не может войти, его refresh-токены отзываются.
- Экспорт CSV защищён от формул Excel (ячейки с `=`, `+`, `-`, `@` в начале получают апостроф).
