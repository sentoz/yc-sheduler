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
yc-scheduler --config config.yaml --sa-key /path/to/sa-key.json

# Базовый запуск с ключом сервисного аккаунта из переменной окружения
export YC_SA_KEY_FILE="/path/to/sa-key.json"
yc-scheduler --config config.yaml

# Запуск с токеном (короткоживущий IAM/OAuth токен, не рекомендуется)
yc-scheduler --config config.yaml --token $(yc iam create-token)

# Режим dry-run (без реальных изменений)
yc-scheduler --config config.yaml --sa-key /path/to/sa-key.json --dry-run

# Запуск с настройкой логирования
yc-scheduler --config config.yaml --sa-key /path/to/sa-key.json --log-level debug --log-format json
```

### Параметры командной строки

- `-c, --config` (обязательно) — путь к конфигурационному файлу
- `--sa-key` — путь к JSON ключу сервисного аккаунта Yandex Cloud
  (можно передать через переменную окружения `YC_SA_KEY_FILE`)
- `-t, --token` (опционально) — IAM/OAuth токен Yandex Cloud
  (переопределяет переменную окружения `YC_TOKEN`, не рекомендуется
  для длительных процессов)
- `-n, --dry-run` — режим тестового запуска без выполнения операций
- `--log-level` — уровень логирования (`trace`, `debug`, `info`, `warn`, `error`)
  (по умолчанию `info`, можно передать через переменную окружения `LOG_LEVEL`)
- `--log-format` — формат логирования (`json` или `console`)
  (по умолчанию `console`, можно передать через переменную окружения `LOG_FORMAT`)

### Переменные окружения

Для удобства можно использовать переменные окружения вместо флагов:

- `YC_SA_KEY_FILE` — путь к файлу ключа сервисного аккаунта
- `YC_TOKEN` — IAM/OAuth токен (не рекомендуется для длительных процессов)
- `LOG_LEVEL` — уровень логирования (`trace`, `debug`, `info`, `warn`, `error`)
- `LOG_FORMAT` — формат логирования (`json` или `console`)

### Конфигурация

Пример конфигурационного файла (`config.yaml`):

```yaml
# Глобальные настройки
timezone: Europe/Moscow              # Таймзона для расписаний (по умолчанию системная)
max_concurrent_jobs: 5               # Максимальное количество одновременных задач (по умолчанию 5)
validation_interval: 10m              # Интервал проверки состояния ресурсов (по умолчанию 10m)
shutdown_timeout: 5m                  # Таймаут graceful shutdown (по умолчанию 5m)
metrics_enabled: false                # Включить Prometheus метрики (по умолчанию false)
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

  # Пример с разными расписаниями для start и stop
  - name: k8s-cluster-maintenance
    type: weekly
    resource:
      type: k8s_cluster
      id: catabcdef1234567890
      folder_id: b1g1234567890abcdef
    actions:
      stop:
        enabled: true
        day: 0  # Sunday
        time: 02:00
      start:
        enabled: true
        day: 1  # Monday
        time: 02:15
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

### Действия

Для каждого ресурса можно настроить действия:

- **start** — запуск ресурса
- **stop** — остановка ресурса

### Метрики Prometheus

При включении метрик (`metrics_enabled: true`) доступны следующие эндпоинты:

- `http://localhost:9090/metrics` — метрики Prometheus
- `http://localhost:9090/health/live` — liveness probe
- `http://localhost:9090/health/ready` — readiness probe
- `http://localhost:9090/` — информационная страница

Метрика `yc_scheduler_operations_total` содержит счетчики операций с лейблами:

- `resource_type` — тип ресурса (vm, k8s_cluster)
- `action` — действие (start, stop)
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
