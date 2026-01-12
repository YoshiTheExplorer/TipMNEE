include .env
export

POSTGRES_CONTAINER=tipmnee-postgres
MIGRATIONS_DIR=db/migrations

postgres:
	docker run --name $(POSTGRES_CONTAINER) \
		-p $(POSTGRES_PORT):5432 \
		-e POSTGRES_USER=$(POSTGRES_USER) \
		-e POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
		-e POSTGRES_DB=$(POSTGRES_DB) \
		-d postgres:12

postgresrm:
	docker rm -f $(POSTGRES_CONTAINER)

createdb:
	docker exec -it postgres12 createdb -U root $(POSTGRES_DB)

dropdb:
	docker exec -it $(POSTGRES_CONTAINER) dropdb -U $(POSTGRES_USER) $(POSTGRES_DB)

createMigrations:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrateup:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_SOURCE)" -verbose up

migratedown:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_SOURCE)" -verbose down

migrateup1:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_SOURCE)" -verbose up 1

migratedown1:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_SOURCE)" -verbose down 1

sqlc:
	sqlc generate

test:
	go test -v -cover ./db/sqlc

server:
	go run main.go

.PHONY: postgres postgresrm dropdb migrateup migratedown sqlc test createMigrations migrateup1 migratedown1 server
