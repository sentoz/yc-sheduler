# yc-scheduler

Утилита для автоматического управления ресурсами Yandex Cloud
(ВМ и Kubernetes) по расписанию.

## Описание

`yc-scheduler` позволяет автоматизировать включение/выключение
виртуальных машин и управление ресурсами Kubernetes в Yandex Cloud по
заданному расписанию.
Поддерживает различные типы планирования:
cron, ежедневные, еженедельные и ежемесячные задачи.

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

# Запуск, когда путь к конфигу передан через переменную окружения
export YC_SHEDULER_CONFIG="config.yaml"
yc-scheduler --sa-key /path/to/sa-key.json

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
  (можно передать через переменную окружения `YC_SHEDULER_CONFIG`)
- `--sa-key` — путь к JSON ключу сервисного аккаунта Yandex Cloud
  (можно передать через переменную окружения `YC_SA_KEY_FILE`)
- `-t, --token` (опционально) — IAM/OAuth токен Yandex Cloud
  (переопределяет переменную окружения `YC_TOKEN`, не рекомендуется
  для длительных процессов)
- `-n, --dry-run` — режим тестового запуска без выполнения операций
- `--version` — вывести информацию о версии и завершить работу
- `--log-level` — уровень логирования (`trace`, `debug`, `info`, `warn`, `error`)
  (по умолчанию `info`, можно передать через переменную окружения `LOG_LEVEL`)
- `--log-format` — формат логирования (`json` или `console`)
  (по умолчанию `console`, можно передать через переменную окружения `LOG_FORMAT`)

### Переменные окружения

Для удобства можно использовать переменные окружения вместо флагов:

- `YC_SHEDULER_CONFIG` — путь к конфигурационному файлу
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
schedules_dir: ./examples/schedules    # Каталог с schedule-манифестами YAML
```

Пример schedule-документа (`examples/schedules/vm-daily.yaml`):

```yaml
apiVersion: scheduler.yc/v1alpha1
kind: Schedule
metadata:
  name: vm-production-workhours
spec:
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
      enabled: true
      time: 18:00
```

Полный пример конфигурации см. в [`config.example.yaml`](config.example.yaml).
Примеры schedule-манифестов см. в [`examples/schedules/`](examples/schedules).
Один YAML-файл может содержать несколько документов через `---`.

### Автоперезагрузка расписаний

Приложение автоматически отслеживает изменения файлов `*.yaml`/`*.yml` в
`schedules_dir`.

- Если обновленные манифесты невалидны, текущие расписания продолжают
  использоваться.
- Если задача уже выполняется в момент изменения расписания, текущий запуск
  не прерывается; изменения применяются только к следующим срабатываниям.

### Развёртывание в Kubernetes

Для развёртывания выполните:

```bash
# (опционально) создать namespace
kubectl create namespace yc-scheduler

# создать Secret с ключом сервисного аккаунта
kubectl -n yc-scheduler create secret generic yc-sa-key \
  --from-file=sa-key.json=/path/to/sa-key.json

# развернуть yc-scheduler
kubectl apply -k deploy/
```

Внутри контейнера:

- конфигурация будет доступна по пути `/config/config.yaml`;
- schedule-манифесты будут доступны по пути `/schedules/*.yaml`;
- ключ сервисного аккаунта — по пути `/sa/sa-key.json`;
- путь до этих файлов также проброшен через переменные окружения
  `YC_SHEDULER_CONFIG` и `YC_SA_KEY_FILE`.

### Типы расписаний

- **daily** — ежедневно в указанное время
- **weekly** — еженедельно в указанный день недели
- **monthly** — ежемесячно в указанный день месяца
- **cron** — по cron-выражению

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
- `http://localhost:9090/` — информация о сборке приложения (JSON с версией,
  коммитом, временем сборки)

Метрика `yc_scheduler_operations_total` содержит счетчики операций с лейблами:

- `resource_type` — тип ресурса (vm, k8s_cluster)
- `action` — действие (start, stop)
- `status` — статус (success, error, dry_run)

### Валидатор состояния

Валидатор периодически проверяет состояние ресурсов и автоматически
исправляет расхождения с расписанием:

- Запускается с интервалом, заданным в `validation_interval` (по умолчанию 10
  минут)
- Определяет ожидаемое состояние ресурса на основе последних времен выполнения
  действий `start` и `stop` из расписания
- Если последнее действие `stop` было позже последнего `start`, ресурс должен
  быть остановлен, и наоборот
- При обнаружении несоответствия создает корректирующую задачу для приведения
  ресурса в ожидаемое состояние
- Пропускает проверку для ресурсов в переходных состояниях (PROVISIONING,
  STOPPING, STARTING и т.д.)

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
