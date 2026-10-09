# Лабораторная работа №2 - Мониторинг сервиса: метрики, логи, трейсы
Условие: https://github.com/KeladKaal/containerization-and-orchestration/blob/main-rus/lecture-2-observability/lab.md


# Ситуация
Нужно поднять полный мониторинг одного сервиса: метрики, логи и трейсы. Метрики и логи смотришь в Grafana, трейсы — в Jaeger, алерты — в Alertmanager. Этот стек соберается один раз дальше будет переиспользоваться.

# Часть 00 — Начальная настройка
1. Нужные программы
![](screenshots/part_0/1.png)

-> Все установлено и готово к работе

2. Создание и настройка кластера kind
![](screenshots/part_0/2.png)

**Анализ:**
- Кластер создан 
- Кластер поднялся, он содежит одну нода Ready и кучу системных подов в kube-system
- kubectl видит кластер kind - контекст kind-monitoring

# Часть 0 - Свой сервис
1. Эндпоинты /health, /fail, /slow, /load

Создание сервиса с четырьмя эндпоинтами. Без метрик, без JSON-логов, без OTel. Просто рабочий HTTP-сервер, который можно дёргать.
![](screenshots/part_0/3.png)

-> Все эндпоинты работают 

2. Метрики Prometheus на /metrics (RED)

1) Подкдючение библиотек
- `go get github.com/prometheus/client_golang/prometheus`
- `go get github.com/prometheus/client_golang/prometheus/promhttp`

2) Создание метрик и middleware
Проверка
![](screenshots/part_0/4.png)

Анализ
- /health → ok, /fail → 500 — эндпоинты живы.
- /metrics отдаёт стандартные Go-метрики: go_gc_duration_seconds, go_goroutines, go_memstats_*. Это автоматически регистрирует prometheus/client_golang — полезно для мониторинга самого сервиса.

- Добавленные RED метрики

![](screenshots/part_0/5.png)

-> Работает, статистика собирается

3. JSON-логи с trace_id

1) fmt.Println / fmt.Printf заменены на slog для того, чтобы формировать записи как структуры: отдельное сообщение, уровень серьёзности и набор атрибутов в виде пар «ключ-значение», и для того, чтобы системам сбора логов - Loki позволить индексировать данные и эффективно с ними работать.
2) Добаялется логирование запросов в middleware
3) Добаялется логирование бизнес-событий
4) Проверка
![](screenshots/part_0/6.png)
-> Всё структурно, парсится, поля корректные. Переход с обычных выводов на Json логи произошел отлично

3. Dockerfile
1) Написать Dockerfile
2) Проверка работы 
![](screenshots/part_0/7.png)

Анализ:
Сервис рабаотет, логи собираются


