package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

// heldMemory хранит выделенные срезы, чтобы GC их не собрал.
// Мьютекс защищает от гонок, если несколько /eat придут одновременно.
var (
	heldMemory [][]byte
	memMu      sync.Mutex
)

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/eat", eatHandler)
	http.HandleFunc("/burn", burnHandler)

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
	fmt.Println("On url /health, ok")
}

// eatHandler выделяет N МБ памяти и держит их до перезапуска.
func eatHandler(w http.ResponseWriter, r *http.Request) {
	mbStr := r.URL.Query().Get("mb")
	mb, err := strconv.Atoi(mbStr)
	if err != nil || mb <= 0 {
		http.Error(w, "invalid mb parameter", http.StatusBadRequest)
		return
	}

	size := mb * 1024 * 1024
	block := make([]byte, size)

	// Записываем в каждый байт, чтобы память реально зафиксировалась
	for i := range block {
		block[i] = 1
	}

	memMu.Lock()
	heldMemory = append(heldMemory, block)
	total := len(heldMemory)
	memMu.Unlock()

	fmt.Fprintf(w, "allocated %d MB, total blocks: %d\n", mb, total)
	fmt.Println("allocated %d MB, total blocks: %d", mb, total)	
}

// burnHandler запускает бесконечный цикл на одном ядре
func burnHandler(w http.ResponseWriter, r *http.Request) {
	go func() {
		var x uint64
		for {
			x++
			_=x
		}
	}()

	fmt.Fprint(w, "burning\n")
	fmt.Println("on URL /burnin, burnin")
}
