# jitLog

Telegram-бот для трекинга тренировок по джиу-джитсу.

## Стек

- Go, Telegram Bot API
- PostgreSQL (pgx), миграции — [goose](https://github.com/pressly/goose)
- Prometheus-метрики

## Запуск

```bash
cp .env.example .env   # указать TELEGRAM_TOKEN и DATABASE_URL
go run ./cmd/service
```

## Миграции

```bash
goose -dir migrations/postgresql/master postgres "$DATABASE_URL" up
```
