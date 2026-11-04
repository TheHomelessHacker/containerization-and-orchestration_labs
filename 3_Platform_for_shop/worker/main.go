package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	log.Println("Worker started. Polling for orders every 5 seconds...")

	for {
		// Ищем необработанные заказы
		rows, err := db.Query("SELECT id, item FROM orders WHERE processed = FALSE LIMIT 10")
		if err != nil {
			log.Println("Query error:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var ids []int
		for rows.Next() {
			var id int
			var item string
			if err := rows.Scan(&id, &item); err == nil {
				ids = append(ids, id)
				log.Printf("Processing order %d: %s\n", id, item)
			}
		}
		rows.Close()

		// Помечаем как обработанные
		for _, id := range ids {
			_, err := db.Exec("UPDATE orders SET processed = TRUE WHERE id = $1", id)
			if err != nil {
				log.Println("Update error:", err)
			}
		}

		time.Sleep(5 * time.Second)
	}
}