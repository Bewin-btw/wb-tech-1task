# Сервис обработки заказов

Микросервис для обработки и отображения данных о заказах с использованием Kafka, PostgreSQL и кэширования.

## Функциональность

- **Прием данных** из Kafka в формате JSON
- **Сохранение данных** в PostgreSQL с использованием транзакций
- **Кэширование данных** в памяти для быстрого доступа
- **REST API** для получения информации о заказах
- **Веб-интерфейс** для просмотра данных о заказах
- **Восстановление кэша** из БД при перезапуске сервиса

## Технологии

- **Go** (язык программирования)
- **PostgreSQL** (база данных)
- **Kafka** (брокер сообщений)
- **Chi** (HTTP-роутер)
- **Zap** (логирование)

## Структура проекта

```
wb-tech-1task/
├── internal/
│   ├── app/          # Основная логика приложения
│   ├── cache/        # Реализация кэширования
│   ├── config/       # Конфигурация приложения
│   ├── db/postgres/  # Работа с PostgreSQL
│   ├── kafka/        # Работа с Kafka
│   ├── models/       # Модели данных
│   ├── server/       # HTTP-сервер и роутинг
│   └── service/      # Бизнес-логика
├── web/static/       # Веб-интерфейс
└── main.go           # Точка входа
```

## Запуск сервиса

### Требования

- Go 1.18+
- PostgreSQL
- Kafka

### Сборка и запуск

```bash
# Устранение возможных проблем с кафкой(роняем все тома:))
docker compose down -v

# Запуск
docker compose up -d --build
```

## API endpoints

### Получить информацию о заказе

```
GET /order?uid=<order_uid>
```

Пример ответа:
```json
{
  "order_uid": "b563feb7b2b84b6test",
  "track_number": "WBILMTESTTRACK",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "b563feb7b2b84b6test",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "WBILMTESTTRACK",
      "price": 453,
      "rid": "ab4219087a764ae0btest",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "test",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "2021-11-26T06:22:19Z",
  "oof_shard": "1"
}
```

### Создать новый заказ

```
POST /order
```

Тело запроса должно содержать JSON с данными заказа(пример можете найти в ./mock-data).

### Health check

```
GET /healthz
```

### Readiness check

```
GET /ready
```

## Веб-интерфейс

После запуска сервиса откройте в браузере http://localhost:8080 для доступа к веб-интерфейсу.

## Особенности реализации

1. **Кэширование**: Данные хранятся в памяти с поддержкой TTL
2. **Восстановление состояния**: При запуске кэш восстанавливается из БД
3. **Обработка ошибок**: Некорректные сообщения отправляются в DLQ
4. **Транзакционность**: Операции с БД выполняются в транзакциях
5. **Валидация**: Входящие данные проверяются на корректность

## Миграции базы данных

Миграции находятся в директории `migrations/` и автоматически применяются при запуске сервиса.
