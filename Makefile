############################
# Stackflow
############################

.DEFAULT_GOAL := help

DOCKER_COMPOSE := docker compose

DB_SERVICE := db

DB_USER := stockflow
DB_NAME := stockflow

.PHONY: help c-up c-build c-down c-stop \
				c-restart c-db c-clean c-stop-all

c-up:
	$(DOCKER_COMPOSE) up -d

c-build:
	$(DOCKER_COMPOSE) up build-d

c-down:
	$(DOCKER_COMPOSE) down

c-stop:
	$(DOCKER_COMPOSE) stop

c-restart:
	$(DOCKER_COMPOSE) restart

c-db:
	$(DOCKER_COMPOSE) exec $(DB_SERVICE) psql -U $(DB_USER) -d $(DB_NAME)

c-clean:
	$(DOCKER_COMPOSE) down -v

c-stop-all:
	docker ps -q | xargs -r docker stop

help:
	@echo "Stockflow - makefile commands"
	@echo ""
	@echo "c-up       "
	@echo "c-build    "
	@echo "c-down     "
	@echo "c-stop     "
	@echo "c-restart  "
	@echo "c-db       "
	@echo "c-clean    "
	@echo "c-stop-all "