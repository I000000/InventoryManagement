.DEFAULT_GOAL := help

PROJECT_NAME := ims

.PHONY: help
help: ## Показать справку по командам
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================
# Docker Commands
# ============================================

DOCKER_COMPOSE := docker compose

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
monitor: ## Запустить Prometheus, Grafana, Loki и Promtail
	$(DOCKER_COMPOSE) up -d prometheus grafana loki promtail

# ============================================
# Kubernetes Commands
# ============================================

.PHONY: k8s-apply
k8s-apply: ## Применить все Kubernetes манифесты
	kubectl apply -f k8s/

.PHONY: k8s-delete
k8s-delete: ## Удалить все Kubernetes ресурсы
	kubectl delete -f k8s/

.PHONY: k8s-restart
k8s-restart: ## Перезапустить все Deployment в кластере
	kubectl get deployments -n inventory-system -o name | xargs kubectl rollout restart -n inventory-system

.PHONY: k8s-status
k8s-status: ## Показать статус всех подов
	kubectl get pods -n inventory-system

.PHONY: k8s-logs-api
k8s-logs-api: ## Логи inventory-api
	kubectl logs -f deployment/inventory-api -n inventory-system

.PHONY: k8s-logs-outbox
k8s-logs-outbox: ## Логи outbox-worker
	kubectl logs -f deployment/outbox-worker -n inventory-system

.PHONY: k8s-logs-stock
k8s-logs-stock: ## Логи stock-worker
	kubectl logs -f deployment/stock-worker -n inventory-system

.PHONY: k8s-logs-kafka
k8s-logs-kafka: ## Логи Kafka
	kubectl logs -f deployment/kafka -n inventory-system

.PHONY: k8s-port-forward-api
k8s-port-forward-api: ## Port-forward для API (8080)
	kubectl port-forward svc/inventory-api 8080:8080 -n inventory-system

.PHONY: k8s-port-forward-frontend
k8s-port-forward-frontend: ## Port-forward для фронтенда (3000)
	kubectl port-forward svc/frontend 3000:3000 -n inventory-system

.PHONY: k8s-port-forward-jaeger
k8s-port-forward-jaeger: ## Port-forward для Jaeger (16686)
	kubectl port-forward svc/jaeger 16686:16686 -n inventory-system

.PHONY: k8s-migrate-postgres
k8s-migrate-postgres: ## Применить миграции PostgreSQL в K8s
	kubectl apply -f k8s/postgres-migrate-job.yaml
	@echo "Waiting for job to complete..."
	@sleep 15
	kubectl logs -f job/postgres-migrate -n inventory-system
	kubectl delete job postgres-migrate -n inventory-system

.PHONY: k8s-migrate-clickhouse
k8s-migrate-clickhouse: ## Применить миграции ClickHouse в K8s
	kubectl apply -f k8s/clickhouse-migrate-job.yaml
	@echo "Waiting for job to complete..."
	@sleep 15
	kubectl logs -f job/clickhouse-migrate -n inventory-system
	kubectl delete job clickhouse-migrate -n inventory-system

.PHONY: k8s-migrate
k8s-migrate: k8s-migrate-postgres k8s-migrate-clickhouse ## Применить все миграции в K8s

.PHONY: k8s-rebuild
k8s-rebuild: ## Пересобрать образы и обновить Deployment (локально)
	@echo "Building images..."
	$(DOCKER_COMPOSE) build
	@echo "Loading images into Minikube..."
	minikube image load inventory-api:latest
	minikube image load stock-worker:latest
	minikube image load outbox-worker:latest
	minikube image load frontend:latest
	minikube image load postgres-migrate:latest
	minikube image load clickhouse-migrate:latest
	@echo "Restarting deployments..."
	$(MAKE) k8s-restart
	@echo "Done! Check status: make k8s-status"

.PHONY: k8s-all
k8s-all: k8s-apply k8s-migrate k8s-status ## Полный деплой в Kubernetes