# Менеджер паролей GophKeeper

- [Описание сервиса](docs/README.md)

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
make build-client-linux-arm64 # => ./bin/gophkeeper-server-client-arm64
```

## Сервер

Управление сервером через docker-compose

```bash
make docker-up
make docker-logs
make docker-clean
make docker-server-build
make docker-server-run
make docker-down
```

Параметры запуска сервера:

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

Пример запуска сервера:

```bash
TODO - пример запска ./bin/gophkeeper-server со всеми возможными флагами
TODO - приер запуска ./bin/gophkeeper-server со всеми возможными env параметрами
```

## Клиент

Параметры запуска клиента:

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
  tui         Start TUI interface
  version     Print version information

Flags:
  -h, --help               help for gk
      --log-level string   log level (debug, info, warn, error) (default "info")

Use "gk [command] --help" for more information about a command.
```

Пример запуска клиента:

```
TODO -  примеры запуска ./bin/gophkeeper-client со всеми возможными парамтерами
```