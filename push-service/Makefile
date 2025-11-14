DB_URL?=postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)
REDIS_URL?=redis://$(REDIS_HOST):$(REDIS_PORT)

.PHONY: run build test clean docker-run migrate-create migrate-up migrate-down swagger docker-compose-up docker-compose-down docker-compose-build

build:
	go build -o bin/push-service ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v

clean:
	rm -rf bin/

docker-build:
	@echo "Building Docker image with BuildKit..."
	@echo "This may take several minutes on first build due to dependency downloads..."
	DOCKER_BUILDKIT=1 docker build -t push-service .

docker-run:
	docker run -p 8080:8080 push-service

swagger:
	@echo "Generating Swagger documentation..."
	@swag init -g cmd/server/main.go -o docs/swagger

docker-compose-build:
	DOCKER_BUILDKIT=1 docker-compose build

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

docker-compose-logs:
	docker-compose logs -f

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $${name}

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down

migrate-force:
	@read -p "Enter version to force: " version; \
	migrate -path migrations -database "$(DB_URL)" force $${version}