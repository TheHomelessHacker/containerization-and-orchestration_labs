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

3. JSON-логи

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
- Обаз создан при помощи Multi-stage: golang:1.25-alpine → scratch и статическая линковка
- Сервис рабаотет, логи собираются

4. Деплой в Kubernetes
1) kind не видит образы из локального Docker — у него свой Docker-демон внутри контейнера. Поэтому образ надо загрузить через 
    `kind load docker-image api-monitoring:latest --name monitoring`
![](screenshots/part_0/8.png)

! -> теперь api-monitoring внутри клстера и kind его видит

5. Helm-чарт для сервиса
1) Создание Helm-чарт для деплоя api в k8s. В нём:
- Deployment — описывает, какой образ, сколько реплик, ресурсы.
- Service — даёт стабильный DNS-адрес для доступа к подам.
2) Описание Helm-чарта
- Chart.yaml — имя, версия, описание чарта.
- values.yaml — параметры, которые можно переопределить при установке.
- templates/deployment.yaml — шаблон Deployment
- templates/service.yaml — шаблон Service:
3) Проверка
![](screenshots/part_0/9.png)
- Синтаксис чарта в норме `0 chart(s) failed`
- YAML выводятся с подставленными значениями

![](screenshots/part_0/10.png)
- У Helm статус "deployed" -  Успех. Релиз установлен, ресурсы созданы.
- `kubectl get pods` — под Running, 1/1 READY, 0 RESTARTS
- сервис api-api с ClusterIP

![](screenshots/part_0/11.png)
- Сервис запущен, логи собираются и выводятся в JSON формете