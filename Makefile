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

.PHONY: migrate
migrate: ## Накатить миграции в ClickHouse (запускается однократно)
	$(DOCKER_COMPOSE) run --rm migrate

.PHONY: restart
restart: down up ## Перезапустить все сервисы

.PHONY: clean
clean: down ## Остановить и удалить контейнеры (volume'ы сохраняются)
	$(DOCKER_COMPOSE) rm -f

.PHONY: start
start: up migrate ## Запустить всё и накатить миграции