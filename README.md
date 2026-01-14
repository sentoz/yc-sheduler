# yc-scheduler

Утилита для автоматического управления ресурсами Yandex Cloud
(ВМ и Kubernetes) по расписанию.

## Описание

`yc-scheduler` позволяет автоматизировать включение/выключение
виртуальных машин и управление ресурсами Kubernetes в Yandex Cloud по
заданному расписанию.
Поддерживает различные типы планирования:
cron, ежедневные, еженедельные, ежемесячные задачи и одноразовое выполнение.

## Использование

### Требования

- Сервисный аккаунт Yandex Cloud и ключ в формате JSON
- (опционально) OAuth/IAM токен Yandex Cloud
  (не рекомендуется для долгоживущих процессов)
- Конфигурационный файл в формате YAML или JSON

### Запуск

```bash
# Базовый запуск с ключом сервисного аккаунта из файла
yc-scheduler --config config.yaml --sa-key-path /path/to/sa-key.json

# Базовый запуск с ключом сервисного аккаунта из переменной окружения
export YC_SERVICE_ACCOUNT_KEY_JSON="$(cat sa-key.json)"
yc-scheduler --config config.yaml --sa-key "$YC_SERVICE_ACCOUNT_KEY_JSON"

# Запуск с токеном (короткоживущий IAM/OAuth токен, не рекомендуется)
yc-scheduler --config config.yaml --token $(yc iam create-token)

# Режим dry-run (без реальных изменений)
yc-scheduler --config config.yaml --sa-key-path /path/to/sa-key.json --dry-run
```

### Параметры командной строки

- `-c, --config` (обязательно) — путь к конфигурационному файлу
- `--sa-key` — содержимое JSON ключа сервисного аккаунта Yandex Cloud
  (можно передать, например, через переменную окружения или heredoc)
- `--sa-key-path` — путь к JSON ключу сервисного аккаунта Yandex Cloud
- `-t, --token` (опционально) — IAM/OAuth токен Yandex Cloud
  (переопределяет переменную окружения `YC_TOKEN`, не рекомендуется
  для длительных процессов)
- `-n, --dry-run` — режим тестового запуска без выполнения операций

### Переменные окружения

Для удобства можно использовать переменные окружения вместо флагов:

- `YC_SERVICE_ACCOUNT_KEY_FILE` или `YC_SA_KEY_FILE` — путь к файлу ключа
- `YC_SERVICE_ACCOUNT_KEY_JSON` или `YC_SA_KEY_JSON` — содержимое JSON ключа
- `YC_TOKEN` — IAM/OAuth токен (не рекомендуется для длительных процессов)

### Конфигурация

Пример конфигурационного файла (`config.yaml`):

```yaml
# Глобальные настройки
timezone: Europe/Moscow              # Таймзона для расписаний (по умолчанию системная)
max_concurrent_jobs: 5               # Максимальное количество одновременных задач (по умолчанию 5)
validation_interval: 10m              # Интервал проверки состояния ресурсов (по умолчанию 10m)
shutdown_timeout: 5m                  # Таймаут graceful shutdown (по умолчанию 5m)
metrics_enabled: true                 # Включить Prometheus метрики (по умолчанию false)
metrics_port: 9090                    # Порт для метрик (по умолчанию 9090)

schedules:
  - name: vm-production-start
    type: daily
    resource:
      type: vm
      id: fhm1234567890abcdef
      folder_id: b1g1234567890abcdef
    actions:
      start:
        enabled: true
        time: 09:00
      stop:
        enabled: false
    daily_job:
      time: 09:00
      timezone: Europe/Moscow
```

Полный пример конфигурации см. в [`config.example.yaml`](config.example.yaml).

### Типы расписаний

- **daily** — ежедневно в указанное время
- **weekly** — еженедельно в указанный день недели
- **monthly** — ежемесячно в указанный день месяца
- **cron** — по cron-выражению
- **duration** — через фиксированный интервал времени
- **one-time** — одноразовое выполнение в указанное время

### Типы ресурсов

- **vm** — виртуальная машина
- **k8s_cluster** — кластер Kubernetes
- **k8s_node_group** — группа нод Kubernetes

### Действия

Для каждого ресурса можно настроить действия:

- **start** — запуск ресурса
- **stop** — остановка ресурса
- **restart** — перезапуск ресурса

### Метрики Prometheus

При включении метрик (`metrics_enabled: true`) доступны следующие эндпоинты:

- `http://localhost:9090/metrics` — метрики Prometheus
- `http://localhost:9090/health/live` — liveness probe
- `http://localhost:9090/health/ready` — readiness probe
- `http://localhost:9090/` — информационная страница

Метрика `yc_scheduler_operations_total` содержит счетчики операций с лейблами:

- `resource_type` — тип ресурса (vm, k8s_cluster, k8s_node_group)
- `action` — действие (start, stop, restart)
- `status` — статус (success, error, dry_run)

## Сборка

Проект использует Makefile для управления сборкой и разработкой.

Быстрый старт:

1. `make init` - инициализация проекта
2. `make build` - сборка бинарника
3. `make check` - перед коммитом запустите полную проверку кода
4. `make release` - сборка для всех платформ

### Переменные сборки

При сборке автоматически заполняются следующие переменные:

- `Version` - версия из git тега
- `Commit` - SHA коммита
- `BuildTime` - время сборки в UTC
- `URL` - URL репозитория
