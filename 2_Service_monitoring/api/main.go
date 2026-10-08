package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
)

// errorCount — простой счётчик ошибок в памяти.
// Позже (в Части 1) он станет Prometheus-метрикой.
var errorCount atomic.Int64

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/fail", failHandler)
	http.HandleFunc("/slow", slowHandler)
	http.HandleFunc("/load", loadHandler)

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}

// GET /health — возвращает ok
func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}

// GET /fail — возвращает 500 и увеличивает счётчик ошибок
func failHandler(w http.ResponseWriter, r *http.Request) {
	errorCount.Add(1)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// GET /slow — спит 1-3 секунды
func slowHandler(w http.ResponseWriter, r *http.Request) {
	delay := time.Duration(1+rand.Intn(3)) * time.Second
	time.Sleep(delay)
	fmt.Fprintf(w, "slept %v\n", delay)
}

// GET /load — делает пачку запросов к самому себе
func loadHandler(w http.ResponseWriter, r *http.Request) {
	// 10 запросов к /health и 5 к /slow — чтобы подскочил RPS
	for i := 0; i < 10; i++ {
		resp, err := http.Get("http://localhost:8080/health")
		if err == nil {
			resp.Body.Close()
		}
	}
	for i := 0; i < 5; i++ {
		resp, err := http.Get("http://localhost:8080/slow")
		if err == nil {
			resp.Body.Close()
		}
	}
	fmt.Fprint(w, "load sent: 10x /health, 5x /slow\n")
}