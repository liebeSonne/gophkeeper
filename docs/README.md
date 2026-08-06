# Описание сервиса GophKeeper

## Архитектура

```mermaid
graph TB
    subgraph Client["Client (CLI / TUI)"]
        CLI["CLI Commands"]
        TUI["TUI Interface"]
        STORAGE["SQLite Store"]
        SYNC["Sync Service"]
    end

    subgraph Server["Server (Go + Chi)"]
        API["REST API Handler"]
        AUTH["Auth Middleware (JWT)"]
        DATA_SVC["Data Service"]
        FILE_SVC["File Service"]
        CRYPTO["Crypto (AES-256-GCM)"]
    end

    subgraph External["External Services"]
        PG[(PostgreSQL)]
        MINIO[(MinIO)]
        VAULT[(Vault KV v2)]
    end

    CLI --> API
    CLI --> STORAGE
    TUI --> API
    TUI --> STORAGE
    SYNC --> API
    SYNC --> STORAGE

    API --> AUTH
    AUTH --> DATA_SVC
    AUTH --> FILE_SVC
    DATA_SVC --> CRYPTO
    FILE_SVC --> CRYPTO
    DATA_SVC --> PG
    FILE_SVC --> PG
    FILE_SVC --> MINIO
    CRYPTO --> VAULT
    CRYPTO -.->|"fallback"| ENV["ENCRYPTION_KEY env"]
```

## Схема базы данных

```mermaid
erDiagram
    users {
        uuid id PK
        varchar-64 login UK
        varchar-256 password
        timestamptz created_at
    }

    refresh_token {
        uuid id PK
        uuid user_id FK
        varchar-64 token_hash
        timestamptz expires_at
        timestamptz revoked_at
        timestamptz created_at
    }

    data {
        uuid id PK
        uuid user_id FK
        smallint type
        bytea payload
        text metadata
        timestamptz created_at
        timestamptz updated_at
    }

    file {
        uuid id PK
        uuid user_id FK
        varchar-1024 name
        varchar-256 mime_type
        bigint size
        int chunks_count
        int status
        timestamptz created_at
        timestamptz updated_at
    }

    file_chunk {
        uuid id PK
        uuid file_id FK
        int chunk_index
        bool uploaded
        timestamptz created_at
    }

    users ||--o{ refresh_token : "has"
    users ||--o{ data : "has"
    users ||--o{ file : "has"
    file ||--o{ file_chunk : "contains"
```

### Описание таблиц

| Таблица | Описание | Ограничения |
|---------|----------|-------------|
| `users` | Пользователи | `login` — UNIQUE |
| `refresh_token` | Refresh-токены | `user_id` → `users` (CASCADE), `revoked_at` — NULL |
| `data` | Зашифрованные записи | `user_id` → `users` (CASCADE), `payload` — BYTEA (AES-GCM) |
| `file` | Метаданные файлов | `user_id` → `users` (CASCADE), `status` — IN_PROGRESS / COMPLETED / FAILED |
| `file_chunk` | Чанки файлов | `file_id` → `file` (CASCADE), UNIQUE(`file_id`, `chunk_index`) |

## API

Полная спецификация REST API: [api/swagger/openapi.yml](../api/swagger/openapi.yml)

## Sequence diagrams

### Регистрация и вход

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant DB as PostgreSQL

    Note over C,DB: Регистрация
    C->>S: POST /api/v1/auth/register
    Note right of C: {login, password}
    S->>DB: SELECT (check login UNIQUE)
    DB-->>S: not found
    S->>DB: INSERT users (bcrypt hash)
    DB-->>S: OK
    S-->>C: 201 Created {id, login, created_at}

    Note over C,DB: Вход
    C->>S: POST /api/v1/auth/login
    Note right of C: {login, password}
    S->>DB: SELECT * FROM users WHERE login
    DB-->>S: user row
    S->>S: bcrypt.CompareHashAndPassword
    S->>S: jwt.Sign (access + refresh)
    S->>DB: INSERT refresh_token (token_hash)
    DB-->>S: OK
    S-->>C: 200 {access_token, refresh_token, expires_in, ...}
```

### Refresh token

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant DB as PostgreSQL

    Note over C,DB: Обновление токена
    C->>S: POST /api/v1/auth/refresh
    Note right of C: {refresh_token}
    S->>S: jwt.Parse (validate signature)
    S->>DB: SELECT WHERE token_hash AND user_id
    DB-->>S: token row
    S->>S: check IsActive (revoked_at == nil, !expired)
    S->>DB: UPDATE revoked_at (revoke old)
    DB-->>S: OK
    S->>S: jwt.Sign (new access + refresh)
    S->>DB: INSERT refresh_token (new hash)
    DB-->>S: OK
    S-->>C: 200 {access_token, refresh_token, ...}
```

### Auto-refresh (клиент)

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    C->>S: GET /api/v1/data/list
    Note right of C: Authorization: Bearer <expired>
    S-->>C: 401 Unauthorized
    C->>C: detect 401 → call RefreshFunc
    C->>S: POST /api/v1/auth/refresh
    S-->>C: 200 {new access_token, ...}
    C->>C: SetAuthToken(new_token)
    C->>S: GET /api/v1/data/list
    Note right of C: Authorization: Bearer <new>
    S-->>C: 200 {items, total, ...}
```

### Data — Create / Get / Update / Delete

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant E as Encryptor
    participant DB as PostgreSQL

    Note over C,DB: Create
    C->>S: POST /api/v1/data
    Note right of C: {data: {type, login, password, ...}}
    S->>E: Encrypt(payload JSON)
    E-->>S: encrypted []byte
    S->>DB: INSERT data (ON CONFLICT DO UPDATE)
    DB-->>S: OK
    S->>E: Decrypt(encrypted)
    E-->>S: decrypted payload
    S-->>C: 201 {data: {id, type, login, password, ...}}

    Note over C,DB: Get
    C->>S: GET /api/v1/data/{id}
    S->>DB: SELECT * FROM data WHERE id AND user_id
    DB-->>S: data row
    S->>E: Decrypt(payload)
    E-->>S: decrypted payload
    S-->>C: 200 {id, type, login, password, ...}

    Note over C,DB: Update
    C->>S: PUT /api/v1/data/{id}
    Note right of C: {data: {type, login, password, ...}}
    S->>DB: SELECT * FROM data WHERE id AND user_id
    DB-->>S: data row (exists)
    S->>E: Encrypt(new payload JSON)
    E-->>S: encrypted []byte
    S->>DB: UPDATE data SET payload, updated_at
    DB-->>S: OK
    S-->>C: 200 {data: {id, type, ...}}

    Note over C,DB: Delete
    C->>S: DELETE /api/v1/data/{id}
    S->>DB: SELECT * FROM data WHERE id AND user_id
    DB-->>S: data row (exists)
    S->>DB: DELETE FROM data WHERE id = ANY(...)
    DB-->>S: OK
    S-->>C: 204 No Content
```

### File Upload (chunked)

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant E as Encryptor
    participant DB as PostgreSQL
    participant M as MinIO

    Note over C,DB: Step 1 — Init
    C->>S: POST /api/v1/file/upload/init
    Note right of C: {name, mime_type, size, chunks_count}
    S->>DB: INSERT file (status=IN_PROGRESS)
    S->>DB: INSERT file_chunk × N (uploaded=false)
    DB-->>S: OK
    S-->>C: 201 {file_id, chunks_count}

    Note over C,M: Step 2 — Upload Chunks
    loop Each chunk
        C->>S: POST /api/v1/file/upload/chunk
        Note right of C: multipart {file_id, chunk_index, data}
        S->>E: Encrypt(chunk data)
        E-->>S: encrypted chunk
        S->>M: PutObject (chunk path)
        M-->>S: OK
        S->>DB: UPDATE file_chunk SET uploaded=true
        DB-->>S: OK
        S-->>C: 200 {file_id, chunk_index, uploaded}
    end

    Note over C,M: Step 3 — Complete
    C->>S: POST /api/v1/file/upload/complete
    Note right of C: {file_id}
    S->>DB: SELECT chunks ORDER BY chunk_index
    DB-->>S: chunk list
    loop Each chunk
        S->>M: GetObject (chunk path)
        M-->>S: encrypted chunk
        S->>S: compose encrypted chunks
    end
    S->>M: PutObject (final file, encrypted)
    M-->>S: OK
    S->>M: RemoveObject (chunk paths) × N
    M-->>S: OK
    S->>DB: UPDATE file SET status=COMPLETED
    DB-->>S: OK
    S-->>C: 200 {fileInfo}
```

### File Download

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant E as Encryptor
    participant DB as PostgreSQL
    participant M as MinIO

    C->>S: GET /api/v1/file/{id}/download
    S->>DB: SELECT * FROM file WHERE id AND user_id
    DB-->>S: file row
    S->>M: GetObject (encrypted file)
    M-->>S: encrypted stream
    S->>E: Decrypt(stream)
    E-->>S: decrypted stream
    S-->>C: 200 (application/octet-stream)
```

### File Delete

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant DB as PostgreSQL
    participant M as MinIO

    C->>S: DELETE /api/v1/file/{id}
    S->>DB: SELECT * FROM file WHERE id AND user_id
    DB-->>S: file row (exists)
    S->>M: RemoveObject (file)
    M-->>S: OK
    S->>DB: DELETE FROM file WHERE id = ANY(...)
    Note right of DB: file_chunk CASCADE
    DB-->>S: OK
    S-->>C: 204 No Content
```

## Типы данных

### LOGIN_PASSWORD

```json
{
  "type": "LOGIN_PASSWORD",
  "login": "user@gmail.com",
  "password": "secret123",
  "metadata": "Gmail account"
}
```

### BANK_CARD

```json
{
  "type": "BANK_CARD",
  "card_number": "**** **** **** 1234",
  "card_holder": "John Doe",
  "card_expiry": "12/25",
  "card_cvv": "123",
  "metadata": "Main debit card"
}
```

### TEXT

```json
{
  "type": "TEXT",
  "text": "Any secret text here",
  "metadata": "Secret note"
}
```

### FILE

```json
{
  "type": "FILE",
  "file_ids": ["<file-uuid-1>", "<file-uuid-2>"],
  "metadata": "Important documents"
}
```

> **Примечание:** Поле `metadata` хранится в открытом виде и используется для поиска. Все остальные поля (`login`, `password`, `card_number`, `text`, `file_ids`) шифруются AES-256-GCM и хранятся в `payload (BYTEA)`.

## Офлайн-режим и синхронизация

Клиент работает в режиме **offline-first** — все данные хранятся в локальном SQLite, синхронизация с сервером происходит в фоне.

### Стратегия синхронизации

| Режим | Описание |
|-------|----------|
| **Фоновая** | Горутина запускает синхронизацию каждые 30 секунд |
| **Явная** | Команда `gk sync` выполняет немедленную синхронизацию |
| **Первый запуск** | При пустом локальном хранилище автоматически загружаются все данные с сервера |

### Статусы синхронизации

| Статус | Описание |
|--------|----------|
| `synced` | Запись синхронизирована с сервером |
| `pending` | Запись создана/изменена локально, ожидает отправки на сервер |
| `deleting` | Запись помечена на удаление, ожидает подтверждения от сервера |
| `conflict` | Обнаружен конфликт (решается в пользу сервера) |

### Push (отправка на сервер)

```mermaid
sequenceDiagram
    participant Sync as Sync Service
    participant L as Local SQLite
    participant S as Server

    Note over Sync,S: Push pending entries
    Sync->>L: DataGetPending()
    L-->>Sync: entries (status=pending/deleting)
    loop Each entry
        alt status = pending (create/update)
            Sync->>S: POST/PUT /api/v1/data
            S-->>Sync: 201/200 {id, ...}
            Sync->>L: DataUpdateSyncStatus(synced, remoteID)
        else status = deleting
            Sync->>S: DELETE /api/v1/data/{remoteID}
            S-->>Sync: 204 No Content
            Sync->>L: DataDelete(localID)
        end
    end
```

### Pull (загрузка с сервера)

```mermaid
sequenceDiagram
    participant Sync as Sync Service
    participant L as Local SQLite
    participant S as Server

    Note over Sync,S: Pull (first launch / empty store)
    Sync->>L: DataCount()
    L-->>Sync: count = 0
    Sync->>S: GET /api/v1/data/list?page=1
    S-->>Sync: {items, total_pages}
    loop Each page
        Sync->>S: GET /api/v1/data/list?page=N
        S-->>Sync: {items, ...}
        loop Each item
            Sync->>L: DataPut(entry, remoteID)
        end
    end
```

### Разрешение конфликтов

При конфликте между локальной и серверной версией:
1. Серверная версия имеет приоритет (**server wins**)
2. Локальная запись заменяется на версию с сервера
3. Статус устанавливается в `synced`
4. Ошибка сохраняется в `error_message` для отладки

### Работа в офлайн-режиме

При отсутствии интернета:
- **Создание** — запись сохраняется в локальное хранилище со статусом `pending`
- **Изменение** — запись обновляется локально, статус меняется на `pending`
- **Удаление** — статус меняется на `deleting` (физическое удаление после подтверждения от сервера)
- **Чтение** — данные читаются из локального хранилища
- При восстановлении соединения изменения автоматически синхронизируются

## Сценарии использования

### Новый пользователь

1. Пользователь получает клиент под свою платформу (Linux, macOS, Windows)
2. Инициализация клиента: `gk init --server-address http://localhost:8080`
3. Регистрация: `gk auth register --login user --password pass`
4. Вход: `gk auth login --login user --password pass`
5. Добавление данных: `gk data create --type LOGIN_PASSWORD --payload '{...}'`
6. Данные шифруются на сервере и сохраняются в PostgreSQL

### Существующий пользователь

1. Пользователь получает клиент под свою платформу
2. Инициализация клиента: `gk init --server-address http://server:8080`
3. Вход: `gk auth login`
4. Токены сохраняются в SQLite на диске клиента
5. При первом запуске автоматически загружаются все данные с сервера
6. Данные хранятся в локальном SQLite (offline-first)
7. Фоновая синхронизация каждые 30 секунд поддерживает актуальность данных
8. Явная синхронизация: `gk sync`
9. При истечении access token клиент автоматически делает refresh

### Параллельные клиенты одного пользователя

- Каждый клиент хранит свои токены в локальном SQLite
- Refresh token на сервере имеет `revoked_at` — при использовании токена предыдущие токены могут быть отозваны
- Несколько клиентов могут работать одновременно, каждый с собственным refresh token
- Данные и файлы привязаны к `user_id` и доступны всем авторизованным клиентам пользователя
