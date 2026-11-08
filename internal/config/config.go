package config

import (
	"errors"   // стандартный: errors.Is(...) сравнивает ошибки, в Load() проверяем «файла .env нет»
	"fmt"      // стандартный: fmt.Errorf("...: %w", err) оборачивает ошибку с пояснением
	"io/fs"    // стандартный: файловая система. Нужна ради fs.ErrNotExist, это ошибка «файл не существует»
	"log/slog" // стандартный: логгер. Нужен ради типа slog.Level (debug, info, warn, error)
	"os"       // стандартный: os.Getenv("ENV_FILE") читает переменную окружения с путём к env-файлу
	"time"     // стандартный: время. Нужен ради time.Duration (10s, 15m, 720h)

	"github.com/joho/godotenv"             // внешний: читает файл .env и кладёт его строки в переменные окружения
	"github.com/kelseyhightower/envconfig" // внешний: берёт переменные окружения и заполняет ими поля структуры по тегам
)

type Config struct {
	App      App      // как запущено приложение: окружение, уровень и файл логов. Читает pkg/logger
	HTTP     HTTP     // порт и таймауты сервера. Читает app.go
	Postgres Postgres // подключение к БД. Читает pkg/postgres
	Redis    Redis    // подключение к Redis (refresh-токены). Читает pkg/redis
	JWT      JWT      // секрет и время жизни токенов. Читает service/auth
}

type App struct {
	Env      string     `envconfig:"APP_ENV" default:"local"`
	LogLevel slog.Level `envconfig:"LOG_LEVEL" default:"info"`
	LogFile  string     `envconfig:"LOG_FILE" default:"app.log"`
}

type HTTP struct {
	// Порт, на котором слушает сервер: http://localhost:8080
	Port string `envconfig:"HTTP_PORT" default:"8080"`

	// ReadTimeout: сколько сервер ждёт, пока клиент ПРИШЛЁТ запрос целиком.
	// Клиент подключился и завис: через 10s сервер закроет соединение.
	ReadTimeout time.Duration `envconfig:"HTTP_READ_TIMEOUT" default:"10s"`

	// WriteTimeout: сколько сервер может ОТВЕЧАТЬ (от получения запроса до отправки ответа).
	// Обработчик завис в долгом SQL: через 10s ответ прервётся.
	WriteTimeout time.Duration `envconfig:"HTTP_WRITE_TIMEOUT" default:"10s"`

	// ShutdownTimeout: сколько сервер ждёт при остановке (Ctrl+C), пока завершатся текущие запросы.
	// Через 10s после Ctrl+C сервер выключится принудительно, даже если запрос не закончился.
	ShutdownTimeout time.Duration `envconfig:"HTTP_SHUTDOWN_TIMEOUT" default:"10s"`
}

type Postgres struct {
	Host string `envconfig:"POSTGRES_HOST" default:"localhost"`

	// required:"true" = обязательная переменная. Если её нет в .env,
	// программа сразу остановится с ошибкой (default не подставляется).
	Port     string `envconfig:"POSTGRES_PORT" required:"true"`
	User     string `envconfig:"POSTGRES_USER" required:"true"`
	Password string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	Database string `envconfig:"POSTGRES_DB" required:"true"`

	SSLMode string `envconfig:"POSTGRES_SSLMODE" default:"disable"` // disable = без шифрования, для локального Docker
}

type Redis struct {
	Host string `envconfig:"REDIS_HOST" default:"localhost"`

	// required:"true" = обязательная переменная: нет в .env, программа остановится с ошибкой.
	Port string `envconfig:"REDIS_PORT" required:"true"`

	Password string `envconfig:"REDIS_PASSWORD"` // без default: пустой пароль допустим
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

type JWT struct {
	// required:"true" = обязательная переменная. Секрет без значения по умолчанию:
	// подписывать токены известной всем строкой нельзя.
	Secret string `envconfig:"JWT_SECRET" required:"true"`

	AccessTTL  time.Duration `envconfig:"JWT_ACCESS_TTL" default:"15m"`
	RefreshTTL time.Duration `envconfig:"JWT_REFRESH_TTL" default:"720h"`
}

func Load() (*Config, error) {
	// 1. Какой env-файл читать. Переменная ENV_FILE хранит путь к файлу ВНЕ проекта
	//    (например C:\Users\shahr\secrets\hr-system.env). ENV_FILE не задана: берём .env в текущей папке.
	path := os.Getenv("ENV_FILE")
	explicit := path != "" // путь указан явно?
	if !explicit {
		path = ".env"
	}

	// 2. Прочитать файл и положить его строки в переменные окружения.
	//    Запасного .env нет (fs.ErrNotExist): не ошибка, переменные могут прийти из системы.
	//    А если путь указан через ENV_FILE и файла нет, молчать нельзя: скорее всего опечатка в пути.
	if err := godotenv.Load(path); err != nil {
		if explicit || !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("load env file %q: %w", path, err)
		}
	}

	// ┌─────────────────────────────────────────────────────────────────────────┐
	// │ envconfig.Process("", &cfg)                                             │
	// ├─────────────────────────────────────────────────────────────────────────┤
	// │ envconfig.Process  главная функция пакета: читает переменные окружения  │
	// │                    и раскладывает их по полям структуры по тегам        │
	// │ ""                 префикс. Пустой: имена как в тегах (HTTP_PORT).      │
	// │                    Если "APP", искал бы APP_HTTP_PORT                   │
	// │ &cfg               адрес переменной cfg: функция заполнит именно        │
	// │                    твою структуру, а не её копию                        │
	// │ err := ...; err != nil                                                  │
	// │                    вызвать, ошибку положить в err; если она не пустая,  │
	// │                    выполнить блок {...}                                 │
	// └─────────────────────────────────────────────────────────────────────────┘
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	return &cfg, nil
}

// ─────────────────────────────────────────────────────────────
// ШПАРГАЛКА: required или default
// ─────────────────────────────────────────────────────────────
// Вопрос: «Если значение забыли, есть ли безопасное значение за всех?»
//
//   Ответ                              Что ставить       Примеры
//   ---------------------------------  ----------------  ------------------------------------
//   Да, безопасное значение есть       default           HTTP_PORT=8080, таймауты 10s,
//                                                        POSTGRES_HOST=localhost, SSLMODE=disable
//   Нет, у каждого своё или опасное    required:"true"   пароли, имя базы, порты баз, JWT_SECRET
//   Может быть пустым                  ничего            REDIS_PASSWORD
//
// ─────────────────────────────────────────────────────────────
// ШПАРГАЛКА: что такое envconfig
// ─────────────────────────────────────────────────────────────
// Это пакет (github.com/kelseyhightower/envconfig) и одновременно тег в структуре.
// Тег — пометка в обратных кавычках после типа поля. Сам Go его не использует,
// его читает пакет envconfig.
//
//   Port string `envconfig:"POSTGRES_PORT" required:"true"`
//                └─ «возьми значение из переменной POSTGRES_PORT, она обязательна»
//
// Для каждого поля envconfig:
//   1. смотрит имя переменной в теге envconfig:"...";
//   2. читает её значение (например, 5432);
//   3. превращает строку в тип поля (string, int, time.Duration, slog.Level);
//   4. кладёт в поле; если значения нет, берёт default, а при required возвращает ошибку.
//
// Теги пишутся через пробел, без двоеточий между парами:
//   `envconfig:"ИМЯ" default:"значение"`
