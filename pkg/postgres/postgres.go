package postgres

import (
	"context" // ctx: управляет временем и отменой операции
	"fmt"     // fmt.Errorf("...: %w", err): оборачивает ошибку с пояснением
	"net"     // net.JoinHostPort: собирает "host:port"
	"net/url" // url.URL: собирает строку подключения, экранирует спецсимволы в пароле

	"github.com/jackc/pgx/v5/pgxpool" // пул соединений с Postgres
)

// Config: настройки подключения. Своя структура («коробка» из шести строк),
// чтобы pkg не зависел от internal/config. Значения в неё копирует app.go.
type Config struct {
	Host     string // где работает база, например localhost
	Port     string // порт базы, например 5433
	User     string // пользователь базы
	Password string // пароль пользователя (в логи не пишем!)
	Database string // имя базы данных
	SSLMode  string // шифрование соединения: disable для локального Docker
}

// NewPool создаёт пул подключений и проверяет, что база отвечает.
//
// Параметры (формат «имя тип»):
//
//	ctx context.Context : контекст операции, переносит отмену и лимит времени
//	cfg Config          : настройки подключения (структура выше)
//
// Возвращает два значения: пул и ошибку. Всё хорошо: (пул, nil). Ошибка: (nil, ошибка).
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	// 1. Создать пул по строке подключения. Эта функция не всегда сразу
	//    подключается к базе, поэтому ниже есть Ping.
	pool, err := pgxpool.New(ctx, cfg.dsn())
	if err != nil {
		// %w кладёт исходную ошибку внутрь новой, её можно проверить через errors.Is
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	// 2. Проверить, что база реально отвечает.
	if err := pool.Ping(ctx); err != nil {
		pool.Close() // не оставляем открытые соединения, если база не отвечает
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	// 3. Всё хорошо: вернуть пул и пустую ошибку.
	return pool, nil
}

// dsn собирает строку подключения через url.URL: спецсимволы в пароле экранируются сами.
// (cfg Config) перед именем значит: это МЕТОД структуры Config, вызывается как cfg.dsn().
// Итог: postgres://user:pass@localhost:5433/hr-db?sslmode=disable
func (cfg Config) dsn() string {
	u := url.URL{
		Scheme:   "postgres",                                    // начало адреса: postgres://
		User:     url.UserPassword(cfg.User, cfg.Password),      // user:pass@, спецсимволы экранируются
		Host:     net.JoinHostPort(cfg.Host, cfg.Port),          // localhost:5433
		Path:     cfg.Database,                                  // /hr-db (слеш добавит u.String() сам)
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}}.Encode(), // ?sslmode=disable
	}
	return u.String() // склеить все части в одну строку
}
