# Лаба 4 — SLA критического пути под перегрузкой

## Условие 
https://github.com/KeladKaal/containerization-and-orchestration/blob/main-rus/lecture-4-scheduling-resources/lab.md

## Ситуация
В shop есть критический путь — оформление заказа (api → postgres) — и второстепенная фоновая нагрузка вроде ночной аналитики, которой под давлением можно пожертвовать. Вся настройка идёт через Helm-чарт shop из Лабы 3: affinity, приоритеты, ресурсы, PodDisruptionBudget — значения и шаблоны чарта, а не разовые kubectl-патчи. Руками только наблюдаешь и провоцируешь нагрузку.

Числа для requests/limits критического пути не придумываются — их даёт krr (Kubernetes Resource Recommender) по реальному потреблению из Prometheus.

# Часть 0 — Сервис batch

## 1. Написание кода для сервиса
## 2. Dockerfile и сборка образа (пока без kind, так как нового кластера еще нет)
![](screenshots/part_0/1.png)
Образ создан