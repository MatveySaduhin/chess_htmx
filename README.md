# ♞ Chess App

Веб-приложение для игры в шахматы в реальном времени с использованием Go, HTMX и WebSockets.

## 🚀 Возможности

- Игра в шахматы в реальном времени
- Аутентификация пользователей и сессии
- Система подбора соперников
- Панель администратора
- Адаптивный дизайн с Tailwind CSS

## 🛠️ Установка

### Требования
- Go 1.21+
- MongoDB

### Локальная разработка

1. **Клонируйте репозиторий**
   ```bash
   git clone <ваш-url-репозитория>
   cd chess-app
   ```
2. **Настройте переменные окружения**
   ```bash
   cp .env.example .env.local
   # Отредактируйте .env.local с вашими значениями
   ```
3. **Запустите приложение**
   ```bash
   go run ./cmd/web
   ```

### Easy Auth integration

For local development with `easy_auth` on `:8080`, run chess on a different port:

```bash
PORT=8081 \
EASY_AUTH_BASE_URL=http://localhost:8080 \
EASY_AUTH_JWKS_URL=http://localhost:8080/.well-known/jwks.json \
EASY_AUTH_ISSUER=easy-auth \
EASY_AUTH_AUDIENCE=easy-auth-api \
go run ./cmd/web
```

The Easy Auth values above are also the chess defaults, except `PORT` defaults to `8080`.

## Тестирование
   ```bash
   # Запустить все тесты
   go test ./...

   # Запустить тесты конкретного пакета
   go test ./internal/game/... -v
   ```
