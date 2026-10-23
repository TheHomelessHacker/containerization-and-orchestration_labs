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

# Часть 1 — Выбор механизма разнесения

## 1. Выбор
- podAntiAffinity (Анти-аффинити) (requiredDuringScheduling + topologyKey): Это классический метод, который жестко разносит поды.Говоришь планировщику: «Не ставь мой под на ноду, если на ней уже есть под с такой-то меткой». У нее есть уровни (required — жестко, preferred — мягко). Это сработает, только если нод много. Если нод 2, а реплик 4 — две реплики останутся в Pending, потому что им физически некуда встать

- topologySpreadConstraints (Ограничения топологии): более умный механизм. Он не запрещает ставить поды на одну ноду, если места больше нет, а просто следит, чтобы перекос перекос между зонами не больше одного пода.

Я выберу для лабораторной `topologySpreadConstraints`? Потому что он решает проблему «асимметрии» при масштабировании. Если будет 3 ноды и 4 реплики, antiAffinity просто раскидает их как получится. А topologySpread сделает так, чтобы разница в количестве подов между нодами была не больше 1 (maxSkew: 1).

## 2. Реализация
1. Перенес структуру чарта из 3 лабы
1. В файлик для управления репликамидобавлена доп настройка `spreadMechanism: "topologySpread"` для api
2. В шаблон для yaml файла был добавлен выбор параметр выбора между podAntiAffinity и topologySpreadConstraints
![](screenshots/part_1/2.png)

# Часть 2 — Тесная площадка

## 1. Создание 2-нодового кластера kind
![](screenshots/part_2/3.png)
![](screenshots/part_2/6.png)
Кластер создан и в нем две ноды `control-plane` и `worker`

## 2. Снятие taint
Был снят taint с control-plane, так как без `kubectl taint nodes lab4-control-plane node-role.kubernetes.io/control-plane:NoSchedule-` поды api не смогли бы планироваться на control-plane, и spread был бы не 2 + 2, а 0 + 4.

## 3. Установка инфраструктуры
### 3.1 Kyverno + policies
Устанавка Kyverno и политик
- Сюда переносяттся настрйоки политик из лабы 3 
![](screenshots/part_2/10.png)
Правила созданы и готовы к работе 

## 3.2 CNPG (postgres-кластер)
`helm upgrade --install cnpg cnpg/cloudnative-pg -n cnpg-system --create-namespace`
![](screenshots/part_2/11.png)
-> Cluster Ready, PVC создан

### 3.3. kube-prometheus-stack (мониторинг)
Будет использоваться только такойже стек, что и в лабе 2
```
helm upgrade --install kps prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace -f helm/monitoring/values.yaml
```

## 4. Сборка и загрузка образов в kind
![](screenshots/part_2/5.png)
Образы попали в кластер

## 5. Добавление шаблона batch в чарт shop
В чарт добавлен templates/batch-deployment.yaml + блок batch в values.yaml

## 6. Деплой чарта shop
`kubectl create namespace shop`
![](screenshots/part_2/8.png)
-> 8 подов shop, shop-api раскидан 2+2 по нодам.

## 7. Проверки состояния
### 7.1 Helm-релизы (вся инфраструктура)
![](screenshots/part_2/7.png)
-> 4 релиза: cnpg, kps, kyverno, shop

### 7.2 Сервисы
![](screenshots/part_2/9.png)

### 7.3 Мониторинг (Prometheus/Grafana/Alertmanager)
![](screenshots/part_2/13.png)
-> видно, что shop-api ServiceMonitor есть и все инструменты готовы к работе

### 7.4 По поводу request/limits 
В Kubernetes класс QoS Guaranteed (Гарантированный) присваивается поду только в том случае, если requests равны limits строго для CPU и для памяти.
Если не поставить CPU limit (следуя лучшей практике с лекции), под получит класс Burstable (как batch). А в Части 7 нам как раз нужно, чтобы api (Guaranteed) не выселялся, а batch (Burstable) выселялся первым. Поэтому в рамках этой лабы CPU limit будут утсановлены так, чтобы получить статус Guaranteed и выполнить задание.
![](screenshots/part_2/12.png)
! все Burstable (в части 5 сделаем api/postgres Guaranteed).