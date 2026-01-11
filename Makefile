postgres:
	docker run --name tipmnee-postgres \
		-p 5434:5432 \
		-e POSTGRES_USER=root \
		-e POSTGRES_PASSWORD=secret \
		-e POSTGRES_DB=tipmnee \
		-d postgres:12

postgresrm:
	docker rm -f tipmnee-postgres

dropdb:
	docker exec -it tipmnee-postgres dropdb -U root tipmnee

createMigrations:
	migrate create -ext sql -dir db/migrations -seq $(name)

migrateup:
	migrate -path db/migrations \
		-database "postgresql://root:secret@localhost:5434/tipmnee?sslmode=disable" \
		-verbose up

migratedown:
	migrate -path db/migrations \
		-database "postgresql://root:secret@localhost:5434/tipmnee?sslmode=disable" \
		-verbose down
		
migrateup1:
	migrate -path db/migrations \
		-database "postgresql://root:secret@localhost:5434/tipmnee?sslmode=disable" \
		-verbose up 1

migratedown1:
	migrate -path db/migrations \
		-database "postgresql://root:secret@localhost:5434/tipmnee?sslmode=disable" \
		-verbose down 1

sqlc:
	sqlc generate

test:
	go test -v -cover ./db/sqlc

server:
	go run main.go

.PHONY: postgres postgresrm dropdb migrateup migratedown sqlc test createMigrations migrateup1 migratedown1 server
