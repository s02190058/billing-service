include .env
export

.PHONY: compose-up
compose-up: ### up docker-compose
	docker-compose up --build

.PHONY: compose-down
compose-down: ### down docker-compose
	docker-compose down --remove-orphans