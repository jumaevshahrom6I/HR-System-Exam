COMPOSE := docker compose -f docker-compose.infra.yaml

.PHONY: run infra-up infra-down infra-reset check build vet test mocks seed

# Запустить приложение (сначала поднимите инфраструктуру: make infra-up)
run:
	go run ./cmd/api

# Поднять Postgres и Redis и дождаться, пока они будут готовы
infra-up:
	$(COMPOSE) up -d --wait

# Остановить. Данные базы сохраняются
infra-down:
	$(COMPOSE) down

# Удалить всё вместе с данными и поднять заново: миграции применятся с нуля
infra-reset:
	$(COMPOSE) down -v
	$(COMPOSE) up -d --wait

# Проверки: сборка, статический анализ и тесты
check: build vet test

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

# Перегенерировать моки репозиториев (после любого изменения интерфейсов в internal/repository/repository.go).
# Нужен mockery: go install github.com/vektra/mockery/v2@v2.43.2
mocks:
	mockery

# Создать первого HR (пароль вводится с клавиатуры)
seed:
	go run ./cmd/seed
