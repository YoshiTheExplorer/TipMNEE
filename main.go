package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/YoshiTheExplorer/TipMNEE/api"
	db "github.com/YoshiTheExplorer/TipMNEE/db/sqlc"
)

func main() {
	_ = godotenv.Load()

	dbSource := os.Getenv("DB_SOURCE")
	if dbSource == "" {
		log.Fatal("DB_SOURCE is required")
	}

	conn, err := sql.Open("postgres", dbSource)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	store := db.New(conn)

	server := api.NewServer(store)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(server.Start(":" + port))
}
