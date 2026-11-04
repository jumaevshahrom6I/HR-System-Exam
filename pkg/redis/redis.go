package redis

import (
	"context" // ctx: управляет временем и отменой операции
	"fmt"     // fmt.Errorf("...: %w", err): оборачивает ошибку с пояснением
	"net"     // net.JoinHostPort: собирает "host:port"

	"github.com/redis/go-redis/v9" // клиент Redis
)

// Config: настройки подключения. Своя структура, чтобы pkg не зависел от internal/config.
// Значения в неё копирует app.go.
type Config struct {
	Host     string // где работает Redis, например localhost
	Port     string // порт Redis, например 6380
	Password string // пароль (локально пустой)
	DB       int    // НОМЕР базы Redis (0, 1, 2...), поэтому int, а не string
}

// NewClient создаёт клиента Redis и проверяет, что сервер отвечает.
// Возвращает клиента и ошибку: всё хорошо (клиент, nil), ошибка (nil, ошибка).
func NewClient(ctx context.Context, cfg Config) (*redis.Client, error) { // ИСПРАВЛЕНО: было New, по ТЗ NewClient
	// 1. Создать клиента. У Redis нет строки подключения, настройки идут в Options.
	client := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(cfg.Host, cfg.Port), // localhost:6380
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 2. Проверить, что сервер отвечает.
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close() // закрываем клиента; ошибку закрытия игнорируем осознанно
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	// 3. Всё хорошо: вернуть клиента и пустую ошибку.
	return client, nil
}
