package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Println("Warning: DB connection failed:", err)
	} else {
		// Создаем таблицу, если ее нет
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			item TEXT NOT NULL,
			processed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			log.Println("Warning: Failed to create table:", err)
		}
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/order", orderHandler)
	http.HandleFunc("/orders", ordersHandler)

	log.Println("API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("HEALTH_FAIL") == "true" {
		http.Error(w, "simulated failure", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Item string `json:"item"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Item == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if db == nil {
		http.Error(w, "DB not ready", http.StatusServiceUnavailable)
		return
	}

	var id int
	err := db.QueryRow("INSERT INTO orders (item) VALUES ($1) RETURNING id", req.Item).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to insert order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id": %d}`, id)
}

func ordersHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "DB not ready", http.StatusServiceUnavailable)
		return
	}

	rows, err := db.Query("SELECT id, item, processed, created_at FROM orders ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Failed to query orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []map[string]interface{}
	for rows.Next() {
		var id int
		var item string
		var processed bool
		var createdAt time.Time
		if err := rows.Scan(&id, &item, &processed, &createdAt); err != nil {
			continue
		}
		orders = append(orders, map[string]interface{}{
			"id":         id,
			"item":       item,
			"processed":  processed,
			"created_at": createdAt,
		})
	}

	json.NewEncoder(w).Encode(orders)
}