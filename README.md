# Менеджер паролей GophKeeper

GophKeeper — клиент-серверная система для хранения зашифрованных паролей, банковских карт, текстовых заметок и файлов.

## Описание сервиса

Подробное описание архитектуры и схем взаимодействия: [docs/README.md](docs/README.md)

## Быстрый старт

### Требования

- Go 1.26+
- Docker + Docker Compose (для запуска зависимостей)

### Запуск сервера

```bash
make docker-server-run
```

Сервер запустится на `http://localhost:8080` с PostgreSQL, Vault и MinIO.

### Настройка клиента

```bash
make build-client
./bin/gophkeeper-client init --server-address http://localhost:8080
./bin/gophkeeper-client auth register --login user --password pass
./bin/gophkeeper-client auth login --login user --password pass
```

## Сервер

### Параметры запуска

```bash
./bin/gophkeeper-server -h
Build version: v1.0.0
Build date: 2026/08/05 23:04:11
Build commit: d9d2dad16e38c63101ce2e4e3cca8968ea79e7af
Usage of server:
      --database-uri string         database connection URI
      --enable-https                enable HTTPS
      --encryption-key string       base64-encoded encryption key (fallback without Vault)
      --jwt-access-ttl duration     JWT access token TTL (default 15m0s)
      --jwt-refresh-ttl duration    JWT refresh token TTL (default 24h0m0s)
      --jwt-secret string           JWT secret key
      --log-level string            log level (debug, info, warn, error) (default "info")
      --server-address string       server address (host:port) (default "0.0.0.0:8080")
      --storage-access-key string   storage access key
      --storage-bucket string       storage bucket name (default "gophkeeper")
      --storage-endpoint string     MinIO/compatible storage endpoint (host:port)
      --storage-secret-key string   storage secret key
      --storage-secure              use HTTPS for storage
      --tls-cert string             path to TLS certificate file
      --tls-key string              path to TLS private key file
      --vault                       enable Vault for encryption key storage
      --vault-address string        Vault address (e.g. http://127.0.0.1:8200)
      --vault-key-path string       Vault KV v2 key path (default "secret/data/gophkeeper/encryption")
      --vault-token string          Vault token
```

### Пример запуска с флагами

```bash
./bin/gophkeeper-server \
  --server-address 0.0.0.0:8080 \
  --log-level debug \
  --database-uri "postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper" \
  --jwt-secret "your-secret-key" \
  --jwt-access-ttl 15m \
  --jwt-refresh-ttl 24h \
  --encryption-key "base64-encoded-32-byte-key" \
  --storage-endpoint localhost:9000 \
  --storage-access-key minioadmin \
  --storage-secret-key minioadmin \
  --storage-bucket gophkeeper \
  --storage-secure=false
```

### Пример запуска с environment variables

```bash
export SERVER_ADDRESS=0.0.0.0:8080
export LOG_LEVEL=debug
export DATABASE_URI="postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper"
export JWT_SECRET="your-secret-key"
export JWT_ACCESS_TTL=15m
export JWT_REFRESH_TTL=24h
export ENABLE_VAULT=false
export VAULT_ADDRESS=http://localhost:8200
export VAULT_TOKEN=dev-token
export VAULT_KEY_PATH=secret/data/gophkeeper/encryption
export ENCRYPTION_KEY="base64-encoded-32-byte-key"
export STORAGE_ENDPOINT=localhost:9000
export STORAGE_ACCESS_KEY=minioadmin
export STORAGE_SECRET_KEY=minioadmin
export STORAGE_BUCKET=gophkeeper
export STORAGE_SECURE=false

./bin/gophkeeper-server
```

### Запуск через Docker

```bash
make docker-up              # Запустить PostgreSQL, Vault, MinIO
make docker-server-build    # Собрать Docker image сервера
make docker-server-run      # Запустить сервер с зависимостями
make docker-logs            # Показать логи
make docker-down            # Остановить все сервисы
make docker-clean           # Остановить и удалить volumes + network
```

## Клиент

### Параметры запуска

```bash
./bin/gophkeeper-client -h
GophKeeper - password manager CLI client

Usage:
  gk [command]

Available Commands:
  auth        Authentication commands
  completion  Generate the autocompletion script for the specified shell
  data        Data management commands
  file        File management commands
  help        Help about any command
  init        Initialize client configuration and storage
  sync        Synchronize local data with server
  tui         Start TUI interface
  version     Print version information

Flags:
  -h, --help               help for gk
      --log-level string   log level (debug, info, warn, error) (default "info")

Use "gk [command] --help" for more information about a command.
```

### Инициализация

```bash
# С параметрами
./bin/gophkeeper-client init --server-address http://localhost:8080 --storage-path ~/.config/gophkeeper/gophkeeper.db

# Со значениями по умолчанию (server: http://localhost:8080, storage: ~/.config/gophkeeper/gophkeeper.db)
./bin/gophkeeper-client init
```

### Аутентификация

```bash
# Регистрация (флаги)
./bin/gophkeeper-client auth register --login user --password pass

# Регистрация (интерактивный режим, bubbletea)
./bin/gophkeeper-client auth register

# Вход (флаги)
./bin/gophkeeper-client auth login --login user --password pass

# Вход (интерактивный режим, bubbletea)
./bin/gophkeeper-client auth login

# Выход
./bin/gophkeeper-client auth logout

# Обновление токена (вручную)
./bin/gophkeeper-client auth refresh
```

### Управление данными

```bash
# Список данных
./bin/gophkeeper-client data list
./bin/gophkeeper-client data list --page 1 --page-size 20
./bin/gophkeeper-client data list --type LOGIN_PASSWORD --search "gmail"
./bin/gophkeeper-client data list --json

# Получить запись по ID
./bin/gophkeeper-client data get --id <uuid>
./bin/gophkeeper-client data get --id <uuid> --json

# Создание записи
./bin/gophkeeper-client data create --type LOGIN_PASSWORD --payload '{"login":"user@gmail.com","password":"secret"}'
./bin/gophkeeper-client data create --type LOGIN_PASSWORD --payload-file payload.json

# Обновление записи
./bin/gophkeeper-client data update --id <uuid> --type LOGIN_PASSWORD --payload '{"login":"new@gmail.com","password":"newpass"}'
./bin/gophkeeper-client data update --id <uuid> --type LOGIN_PASSWORD --payload-file payload.json

# Удаление записи
./bin/gophkeeper-client data delete --id <uuid>
```

### Управление файлами

```bash
# Загрузка файла
./bin/gophkeeper-client file upload --file /path/to/file.pdf

# Скачивание файла
./bin/gophkeeper-client file download --id <uuid> --output /path/to/output.pdf
./bin/gophkeeper-client file download --id <uuid>  # вывод в stdout

# Список файлов
./bin/gophkeeper-client file list
./bin/gophkeeper-client file list --page 1 --page-size 20 --search "report"
./bin/gophkeeper-client file list --json

# Удаление файла
./bin/gophkeeper-client file delete --id <uuid>
```

### Синхронизация

```bash
# Явная синхронизация
./bin/gophkeeper-client sync
```

Клиент поддерживает офлайн-режим с автоматической синхронизацией:
- **Фоновая синхронизация** — каждые 30 секунд (при наличии интернета)
- **Явная синхронизация** — команда `gk sync`
- **Первый запуск** — автоматическая загрузка всех данных с сервера
- **Офлайн-режим** — создание/изменение/удаление записей в локальном хранилище
- **Разрешение конфликтов** — версия с сервера имеет приоритет

### TUI режим

```bash
./bin/gophkeeper-client tui
```

Интерактивный интерфейс с поддержкой:
- Навигации между данными и файлами
- Поиска и фильтрации по типу
- Копирования значений в буфер обмена
- Создания и редактирования записей

## Типы данных

### LOGIN_PASSWORD

```json
{
  "login": "user@gmail.com",
  "password": "secret123"
}
```

### BANK_CARD

```json
{
  "cardNumber": "**** **** **** 1234",
  "cardHolder": "John Doe",
  "cardExpiry": "12/25"
}
```

### TEXT

```json
{
  "text": "Any secret text here"
}
```

### FILE

```json
{
  "fileIds": ["<file-uuid-1>", "<file-uuid-2>"]
}
```

## Makefile

В Makefile вынесены основные команды управления проектом

```bash
make help

Usage:
  make <target>

Targets:
  build-all        Build all platforms
  build-client     Build client binary
  build-server     Build server binary
  clean            Remove binaries
  cover            Run tests coverage
  create-migration  Run create migration
  docker-clean     Stop and remove volumes
  docker-down      Stop all services
  docker-logs      Show logs
  docker-server-build  Build server Docker image
  docker-server-run  Run server with dependencies
  docker-up        Start dependencies (PostgreSQL, Vault, MinIO)
  generate-api-server  Run generate openapi server
  generate-client  Run generate openapi client
  generate         Generate all API code
  help             Show available targets
  lint             Run linter
  mocks            Run generate mocks
  test             Run tests
```

## Сборка

Сборка под разные OS

```bash
# По умолчанию:
make build        # => ./bin/gophkeeper-server, ./bin/gophkeeper-client
make build-server # => ./bin/gophkeeper-server
make build-client # => ./bin/gophkeeper-client
# Все настроенные OS:
make build-all    # => ./bin/gophkeeper-server-windows-amd64
                  # => ./bin/gophkeeper-server-linux-arm64
                  # => ./bin/gophkeeper-client-windows-amd64
                  # => ./bin/gophkeeper-server-linux-amd64
                  # => ./bin/gophkeeper-server-darwin-arm64
                  # => ./bin/gophkeeper-client-linux-arm64
                  # => ./bin/gophkeeper-server-darwin-amd64
                  # => ./bin/gophkeeper-client-darwin-amd64
                  # => ./bin/gophkeeper-client-linux-amd64
                  # => ./bin/gophkeeper-client-darwin-arm64
# Конкретную OS:
make build-server-linux-arm64 # => ./bin/gophkeeper-server-linux-arm64
make build-client-linux-arm64 # => ./bin/gophkeeper-client-linux-arm64
```

## CI/CD

### CI (`.github/workflows/ci.yml`)

- Lint (`golangci-lint`)
- Тесты на Go 1.25 и 1.26
- Сборка бинарников

### Release (`.github/workflows/release.yml`)

- Сборка под 5 платформ: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64
- SHA256SUMS чексуммы
- GitHub Release с бинарниками

### Docker (`.github/workflows/docker.yml`)

- Multi-arch Docker image (linux/amd64, linux/arm64)
- Публикация в GitHub Container Registry (ghcr.io)

