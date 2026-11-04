package main

import (
	"context"
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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var errorCount atomic.Int64
var logger = slog.New(&otelHandler{inner: slog.NewJSONHandler(os.Stdout, nil)})

type otelHandler struct {
	inner slog.Handler
}

func (h *otelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		sc := span.SpanContext()
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{inner: h.inner.WithGroup(name)}
}

// === МЕТРИКИ ===
var (
	// CounterVec для http_requests_total с метками: method, path, status - READE
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"}, // метки
	)

	// HistogramVec для http_request_duration_seconds с метками: method, path
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
	// Регистрация метрик в prometheus
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// === MIDDLEWARE ===
// Оборачивает хендлер: считает время, записывает метрики 
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Обёртка над w, чтобы перехватить status code
		rec := &statusRecorder{ResponseWriter: w, status: 200}

		// Передаю обёртку в хендлер — он пишет через неё
		next(rec, r)

		logger.InfoContext(r.Context(),
			"http request",
			"method", r.Method,
			"path",   r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(rec.status)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	}
}

// === OTel init ===
func initTracer(ctx context.Context) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		logger.Info("OTEL_EXPORTER_OTLP_ENDPOINT not set, tracing disabled")
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("api"),
		)),
	)
	otel.SetTracerProvider(tp)

	logger.Info("tracing enabled", "endpoint", endpoint)
	return tp.Shutdown, nil
}

func main() {

	ctx := context.Background()

	shutdown, err := initTracer(ctx)
	if err != nil {
		logger.Error("tracer init failed", "err", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	http.Handle("/health", otelhttp.NewHandler(metricsMiddleware(healthHandler), "GET /health"))
	http.Handle("/fail", otelhttp.NewHandler(metricsMiddleware(failHandler), "GET /fail"))
	http.Handle("/slow", otelhttp.NewHandler(metricsMiddleware(slowHandler), "GET /slow"))
	http.Handle("/load", otelhttp.NewHandler(metricsMiddleware(loadHandler), "GET /load"))

	// добавлен эндпоинт /metrics через promhttp.Handler()
	http.Handle("/metrics", promhttp.Handler())

	logger.Info("server starting", "port", 8080)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server error", "err", err)
	}
}

// ХЕНДЛЕРЫ
func healthHandler(w http.ResponseWriter, r *http.Request) {
	logger.InfoContext(r.Context(), "health check")
	fmt.Fprintln(w, "ok")
}

// GET /fail — возвращает 500 и увеличивает счётчик ошибок
func failHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("api").Start(r.Context(), "fail-op")
	defer span.End()

	errorCount.Add(1)
	span.SetStatus(codes.Error, "simulated failure")
	span.RecordError(fmt.Errorf("fail endpoint triggered"))

	logger.WarnContext(ctx, "fail endpoint triggered", "total_errors", errorCount.Load())
	http.Error(w, "internal server error", http.StatusInternalServerError)

}

// GET /slow — спит 1-3 секунды
func slowHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("api").Start(r.Context(), "slow-op")
	defer span.End()

	delay := time.Duration(1+rand.Intn(3)) * time.Second
	logger.InfoContext(ctx, "slow started", "delay_s", delay.Seconds())
	time.Sleep(delay)
	logger.InfoContext(ctx, "slow finished", "delay_s", delay.Seconds())

	fmt.Fprintf(w, "slept %v\n", delay)
}

// GET /load — делает пачку запросов к самому себе
func loadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger.InfoContext(ctx, "load started")

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

	logger.InfoContext(ctx, "load finished")
	fmt.Fprint(w, "load sent: 10x /health, 5x /slow\n")
}