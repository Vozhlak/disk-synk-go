# disk-sync-go

**disk-sync-go** — консольная утилита на Go для синхронизации файлов из локальной папки с Google Drive и Яндекс.Диском.

Приложение сравнивает содержимое локальной директории с файлами в облачных хранилищах, определяет файлы, которых нет в облаке, и загружает их параллельно.

## Возможности

- Чтение конфигурации из `config.yaml`
- Структурированное логирование:
    - текстовый формат для `local`
    - JSON-формат для `dev` и `prod`
- OAuth 2.0 авторизация в Google Drive
- OAuth 2.0 авторизация в Яндекс.Диске
- Получение списка файлов из:
    - локальной файловой системы
    - Google Drive
    - Яндекс.Диска
- Единый интерфейс `storage.Storage` для разных хранилищ
- Расчёт списка файлов, отсутствующих в облаке
- Параллельная загрузка файлов с использованием `sync.WaitGroup`
- Потоковая передача содержимого файлов через `io.Reader`

## Требования

- Go **1.27+**
- Google Cloud project с включённым Google Drive API
- OAuth-клиент Google для Web application
- Яндекс OAuth-приложение с доступом к Яндекс.Диск REST API
- Локальная директория с файлами для синхронизации

## Установка

Клонируй репозиторий:

```bash
git clone https://github.com/Vozhlak/disk-synk-go.git
cd disk-sync-go
```

Установи зависимости:

```bash
go mod tidy
```

## Настройка

Создай `config.yaml` в корне проекта:

```yaml
env: local

local_dir: "./sync-source"

google_drive:
  client_id: "your-google-client-id"
  client_secret: "your-google-client-secret"
  token_path: "google_token.json"

yandex_disk:
  client_id: "your-yandex-client-id"
  client_secret: "your-yandex-client-secret"
  token_path: "yandex_token.json"
```

Создай папку с файлами для синхронизации:

```bash
mkdir sync-source
```

Положи в неё файлы, которые нужно загрузить в облака.

> **Важно:** не указывай в `local_dir` корень проекта. Иначе в синхронизацию могут попасть `config.yaml`, OAuth-токены, `.git`, `.idea` и другие служебные файлы.

## Конфигурация

| Поле | Описание |
|---|---|
| `env` | Окружение: `local`, `dev` или `prod` |
| `local_dir` | Путь к локальной папке-источнику файлов |
| `google_drive.client_id` | OAuth Client ID Google |
| `google_drive.client_secret` | OAuth Client Secret Google |
| `google_drive.token_path` | Путь к файлу сохранённого Google OAuth-токена |
| `yandex_disk.client_id` | OAuth Client ID Яндекса |
| `yandex_disk.client_secret` | OAuth Client Secret Яндекса |
| `yandex_disk.token_path` | Путь к файлу сохранённого Яндекс OAuth-токена |

### Google Drive

1. Создай проект в [Google Cloud Console](https://console.cloud.google.com/).
2. Включи **Google Drive API**.
3. Создай OAuth Client ID типа **Web app**.
4. Скопируй `client_id` и `client_secret` в `config.yaml`.
5. При первом запуске приложение откроет ссылку авторизации и запустит локальный callback-сервер.

### Яндекс.Диск

1. Создай приложение в [Яндекс OAuth](https://oauth.yandex.ru/).
2. Добавь права:
    - `cloud_api:disk.read`
    - `cloud_api:disk.write`
3. Используй платформу **Веб-сервисы**.
4. Redirect URI будет фиксированным:

   ```text
   https://oauth.yandex.ru/verification_code
   ```

5. Скопируй `client_id` и `client_secret` в `config.yaml`.
6. При первом запуске приложение выведет ссылку авторизации и попросит вставить код подтверждения из браузера.

## Использование

Запусти приложение:

```bash
go run ./src/cmd/disksync/main.go
```

Или с детектором гонок:

```bash
go run -race ./src/cmd/disksync/main.go
```

### Первый запуск

При первом запуске приложение:

1. Загрузит `config.yaml`.
2. Авторизуется в Google Drive.
3. Авторизуется в Яндекс.Диске.
4. Получит списки файлов из всех хранилищ.
5. Рассчитает файлы, которых нет в облаках.
6. Загрузит их параллельно.

Пример вывода:

```text
level=INFO msg="configuration loaded successfully" path=config.yaml env=local
level=INFO msg="Google Drive client initialized"
Authenticated as: user@example.com
level=INFO msg="Google Drive access check completed successfully"
level=INFO msg="Yandex Disk client initialized"
level=INFO msg="successfully authenticated with Yandex Disk"
level=INFO msg="found local files" count=2
level=INFO msg="calculating diff..." storage="Google Drive"
level=INFO msg="diff calculated" storage="Google Drive" files_to_sync=2
level=INFO msg="uploading file..." storage="Google Drive" file_name=report.docx
level=INFO msg="file upload completed" storage="Google Drive" file_name=report.docx duration=820ms
level=INFO msg="file synchronization completed" duration=1.4s
```

### Повторный запуск

OAuth-токены сохраняются в файлах, указанных в `token_path`. При повторном запуске приложение использует сохранённые токены и не запрашивает авторизацию повторно.

## Как работает синхронизация

1. `localfs` читает файлы из `local_dir`.
2. `gdrive` и `ydisk` получают списки файлов из облаков.
3. `differ.Diff` сравнивает локальные файлы с файлами каждого облака.
4. Файлы, отсутствующие в облаке, добавляются в список для загрузки.
5. Для каждого файла запускается отдельная горутина.
6. `sync.WaitGroup` ожидает завершения всех загрузок.
7. В конце выводится общее время синхронизации.

Локальная папка выступает источником. Приложение загружает файлы, которых нет в облаке, но пока не удаляет файлы, отсутствующие локально.

## Безопасность

Файлы с секретами и OAuth-токенами не должны попадать в Git.

Проверь, что `.gitignore` содержит:

```gitignore
config.yaml
*token.json
```
