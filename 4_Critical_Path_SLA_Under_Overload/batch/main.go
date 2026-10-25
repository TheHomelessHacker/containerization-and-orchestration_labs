package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Batch worker started...")

	// Переменные для контроля потребления (будем менять через ENV)
	cpuIntensity := 1000
	memMB := 10

	if val := os.Getenv("CPU_INTENSITY"); val != "" {
		fmt.Sscanf(val, "%d", &cpuIntensity)
	}
	if val := os.Getenv("MEM_MB"); val != "" {
		fmt.Sscanf(val, "%d", &memMB)
	}

	// 1. Выделяем и держим память (чтобы GC не забрал)
	var heldMemory [][]byte
	for i := 0; i < memMB; i++ {
		block := make([]byte, 1024*1024)
		for j := range block {
			block[j] = 1 // форсируем физическое выделение
		}
		heldMemory = append(heldMemory, block)
	}
	fmt.Printf("Allocated and holding %d MB\n", memMB)

	// 2. Бесконечный цикл, жгущий CPU
	var x uint64
	for {
		for i := 0; i < cpuIntensity; i++ {
			x++
			_ = x
		}
		time.Sleep(10 * time.Millisecond) // чуть спим, чтобы не повесить систему полностью
	}
}