.PHONY: 

all: build

DCE= docker-compose

build:
	$(DCE) up -d --build

up:
	$(DCE) up -d

down:
	$(DCE) down

migrate:
	$(DCE) up migrate -d

logs:
	$(DCE) logs -f app

lint:
	golangci-lint run