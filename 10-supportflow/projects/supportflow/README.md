# SupportFlow — Поддержка с AI-помощником

Проект №10. Самостоятельное Go-приложение. Сборка: `./cmd/supportflow`.
Общие пакеты находятся в `internal/platform`, поэтому сохраняйте структуру репозитория.

## Реализовано в v0.1

Тикеты; SLA-дедлайн; классификация; AI-черновик через LangGraph или явно обозначенный шаблон; admin approval; одно сообщение в локальном inbox; аудит; transactional outbox.

Данные: `tickets, ticket_audit, support_replies`. Схема: `../../migrations/supportflow_001.sql`.
Все записи изолированы по tenant; роль определяется серверным API-токеном.

## Запуск

Из корня репозитория:

```bash
python3 scripts/setup.py  # один раз для всего набора
PROJECT=supportflow docker compose up --build -d app
python3 scripts/demo.py supportflow
```

Для JobRunner запускайте ещё `--profile jobs`; точные команды профилей — в корневом README.
Демо создаёт тестовые записи. Остановить: `docker compose down`.

## API

Все пути требуют `Authorization: Bearer <token>`. Ошибки возвращаются как JSON.
`Idempotency-Key` необязателен, но рекомендуется для команд создания и отправки.
Повтор того же запроса с тем же ключом возвращает сохранённый ответ; изменение запроса даёт 409.

| Метод | Путь |
|---|---|
| POST | `/tickets` |
| GET | `/tickets` |
| GET | `/tickets/{id}` |
| POST | `/tickets/{id}/draft` |
| POST | `/tickets/{id}/approve` |
| POST | `/tickets/{id}/send` |
| GET | `/tickets/{id}/reply` |
| GET | `/tickets/{id}/audit` |

## Примеры тел запросов

`POST /tickets`:

```json
{
  "subject": "Оплата",
  "message": "Не прошла оплата"
}
```

`POST /tickets/{id}/approve`:

```json
{
  "expected_version": 2,
  "reply": "Уточните номер заказа."
}
```

## Проверка

```bash
TEST_DATABASE_URL='postgres://portfolio:portfolio-local-only@localhost:5432/portfolio?sslmode=disable' \
  go test -count=1 -v ./tests -run '^TestSupportFlow$'
```

Интеграционные тесты пишут данные в указанную БД. Используйте отдельную тестовую базу.
Полный сценарий проверки находится в `../../tests/integration_test.go`.

## Границы версии и дальнейшая работа

Назначение исполнителей, поиск знаний и инструмент статуса заказа, полноценные checkpoints LangGraph между его узлами. Сейчас долговечное состояние сохраняется между бизнес-этапами в Go/PostgreSQL; внешние сообщения не отправляются.

Это реализованное ядро проекта, а не заявление о завершении всех расширений исходной идеи.
Подробнее об ограничениях окружения проверки — `../../VALIDATION.md`.
