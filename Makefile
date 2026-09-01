.DEFAULT_GOAL := help

DOCKER_COMPOSE := docker compose
PROJECT_NAME := ims

.PHONY: help
help: ## Показать справку по командам
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Собрать все Docker-образы
	$(DOCKER_COMPOSE) build

.PHONY: up
up: ## Запустить все сервисы в фоне (без миграций)
	$(DOCKER_COMPOSE) up -d

.PHONY: down
down: ## Остановить все сервисы
	$(DOCKER_COMPOSE) down

.PHONY: logs
logs: ## Показать логи всех контейнеров (в реальном времени)
	$(DOCKER_COMPOSE) logs -f

.PHONY: logs-api
logs-api: ## Показать логи API-сервиса
	$(DOCKER_COMPOSE) logs -f inventory-api

.PHONY: logs-worker
logs-worker: ## Показать логи воркера
	$(DOCKER_COMPOSE) logs -f stock-worker

.PHONY: migrate-clickhouse
migrate-clickhouse: ## Применить ClickHouse миграции
	$(DOCKER_COMPOSE) run --rm clickhouse-migrate

.PHONY: migrate-postgres
migrate-postgres: ## Применить PostgreSQL миграции
	$(DOCKER_COMPOSE) run --rm postgres-migrate

.PHONY: migrate
migrate: migrate-postgres migrate-clickhouse ## Применить все миграции

.PHONY: restart
restart: down up ## Перезапустить все сервисы

.PHONY: clean
clean: down ## Остановить и удалить контейнеры (volume'ы сохраняются)
	$(DOCKER_COMPOSE) rm -f

.PHONY: clean-all
clean-all: down ## Остановить всё и удалить volumes
	$(DOCKER_COMPOSE) --profile loadtest down -v

.PHONY: start
start: up migrate monitor ## Запустить всё (сервисы + мониторинг)

.PHONY: loadtest
loadtest: ## Запустить нагрузочный тест (k6) – пересобирает образ
	$(DOCKER_COMPOSE) --profile loadtest build k6
	$(DOCKER_COMPOSE) --profile loadtest up k6

.PHONY: monitor
monitor: ## Запустить Prometheus и Grafana
	$(DOCKER_COMPOSE) up -d prometheus grafana