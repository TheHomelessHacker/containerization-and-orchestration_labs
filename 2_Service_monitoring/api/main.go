package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
	"strconv"
	"log/slog"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var errorCount atomic.Int64
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))


// === МЕТРИКИ ===
var (
	// TODO: создай CounterVec для http_requests_total с метками: method, path, status - READE
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"}, // ← метки
	)

	// TODO: создай HistogramVec для http_request_duration_seconds с метками: method, path - READE
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 3, 5}, // ← бакеты
		},
		[]string{"method", "path"},
	)
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func init() {
	// TODO: зарегистрируй метрики в prometheus - READE
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// === MIDDLEWARE ===
// Оборачивает хендлер: считает время, записывает метрики 
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// TODO: создай обёртку над w, чтобы перехватить status code
		// (стандартный http.ResponseWriter не даёт узнать status после WriteHeader)
		// Подсказка: struct { http.ResponseWriter; status int } - READE
		rec := &statusRecorder{ResponseWriter: w, status: 200}

		// Передаём обёртку в хендлер — он пишет через неё - READE
		next(rec, r)

		logger.Info("http request",
			"method", r.Method,
			"path",   r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		// TODO: посчитать duration, записать в histogram
		// TODO: записать в counter с метками method, path, status
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(rec.status)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	}
}

func main() {
	http.HandleFunc("/health", metricsMiddleware(healthHandler))
	http.HandleFunc("/fail", metricsMiddleware(failHandler))
	http.HandleFunc("/slow", metricsMiddleware(slowHandler))
	http.HandleFunc("/load", metricsMiddleware(loadHandler))

	// TODO: добавь эндпоинт /metrics через promhttp.Handler()
	http.Handle("/metrics", promhttp.Handler())

	logger.Info("server starting", "port", 8080)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server error", "err", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}

// GET /fail — возвращает 500 и увеличивает счётчик ошибок
func failHandler(w http.ResponseWriter, r *http.Request) {
	errorCount.Add(1)
	logger.Warn("fail endpoint triggered", "total_errors", errorCount.Load())
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// GET /slow — спит 1-3 секунды
func slowHandler(w http.ResponseWriter, r *http.Request) {
	delay := time.Duration(1+rand.Intn(3)) * time.Second
	logger.Info("slow request started", "delay_seconds", delay.Seconds())
	time.Sleep(delay)
	logger.Info("slow request finished", "delay_seconds", delay.Seconds())
	fmt.Fprintf(w, "slept %v\n", delay)
}

// GET /load — делает пачку запросов к самому себе
func loadHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("load started")
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
	logger.Info("load finished")
	fmt.Fprint(w, "load sent: 10x /health, 5x /slow\n")
}