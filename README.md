# 🛡️ Phishing Monitor

Сервис мониторинга фишинговых доменов для SOC аналитиков. Автоматически отслеживает новые SSL сертификаты через Certstream и анализирует домены на схожесть с доменами ваших клиентов.

## 🚀 Возможности

- **Мониторинг в реальном времени** через Certstream API
- **Интеллектуальный анализ схожести** доменов с использованием алгоритма Левенштейна
- **Интеграция с открытыми базами фишинга**:
  - NRD (Newly Registered Domains)
  - Phishing Database
  - Hagezi DNS Blocklists
- **Telegram уведомления** с детальной информацией
- **Ежедневные отчеты** по обнаруженным угрозам
- **SQLite база данных** для хранения результатов
- **Docker поддержка** для простого развертывания

## 📋 Требования

- Go 1.21+
- SQLite3
- Telegram Bot Token

## 🔧 Установка

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd phishing-monitor
```

### 2. Настройка конфигурации

Скопируйте и отредактируйте файл конфигурации:

```bash
cp .env.example .env
```

Заполните необходимые переменные в `.env`:

```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_CHAT_ID=your_chat_id_here
CLIENT_DOMAINS=,yourcompany.com,client.org
SIMILARITY_THRESHOLD=0.8
```

### 3. Установка зависимостей

```bash
go mod download
```

### 4. Запуск

```bash
go run cmd/monitor/main.go
```

## 🐳 Docker развертывание

### Быстрый старт с Docker Compose

```bash
# Настройте .env файл
cp .env.example .env
# Отредактируйте .env с вашими настройками

# Запуск
docker-compose up -d
```

### Ручная сборка Docker образа

```bash
docker build -t phishing-monitor .
docker run -d --name phishing-monitor \
  -e TELEGRAM_BOT_TOKEN=your_token \
  -e TELEGRAM_CHAT_ID=your_chat_id \
  -e CLIENT_DOMAINS=,example.com \
  -v $(pwd)/data:/app/data \
  phishing-monitor
```

## ⚙️ Конфигурация

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `TELEGRAM_BOT_TOKEN` | Токен Telegram бота | - |
| `TELEGRAM_CHAT_ID` | ID чата для уведомлений | - |
| `CLIENT_DOMAINS` | Домены клиентов (через запятую) | `,example.com` |
| `CERTSTREAM_URL` | URL Certstream WebSocket | `wss://certstream.calidog.io` |
| `DATABASE_PATH` | Путь к SQLite базе | `./data/phishing.db` |
| `LOG_LEVEL` | Уровень логирования | `info` |
| `UPDATE_INTERVAL` | Интервал обновления баз (мин) | `60` |
| `SIMILARITY_THRESHOLD` | Порог схожести (0.0-1.0) | `0.8` |

### Создание Telegram бота

1. Найдите [@BotFather](https://t.me/botfather) в Telegram
2. Отправьте `/newbot` и следуйте инструкциям
3. Получите токен бота
4. Добавьте бота в чат и получите Chat ID

Для получения Chat ID:
```bash
curl "https://api.telegram.org/bot<BOT_TOKEN>/getUpdates"
```

## 📊 Как это работает

### Алгоритм обнаружения

1. **Получение данных**: Подключение к Certstream для получения новых SSL сертификатов
2. **Извлечение доменов**: Парсинг CN и SAN полей сертификатов
3. **Анализ схожести**: Сравнение с доменами клиентов используя:
   - Расстояние Левенштейна
   - Проверка подстрок
   - Анализ фишинговых ключевых слов
4. **Проверка в базах**: Сопоставление с базами известных фишинговых доменов
5. **Уведомления**: Отправка алертов в Telegram с рекомендациями

### Источники данных фишинга

- **NRD**: Новые зарегистрированные домены с DGA паттернами
- **Phishing Database**: Актуальная база фишинговых доменов
- **Hagezi**: DNS блок-листы

## 📱 Telegram команды

- `/start` - Активация бота
- `/status` - Проверка статуса мониторинга

## 📈 Мониторинг и отчеты

### Ежедневные отчеты

Автоматически отправляются в 9:00 UTC и включают:
- Количество обнаруженных доменов
- Статистику по клиентам
- Домены высокого риска
- Тренды и рекомендации

### Уровни риска

- 🔴 **HIGH**: Схожесть >95% или содержит фишинговые ключевые слова
- 🟡 **MEDIUM**: Схожесть 85-95%
- 🟢 **LOW**: Схожесть 80-85%

## 🔧 Разработка

### Структура проекта

```
phishing-monitor/
├── cmd/monitor/          # Точка входа приложения
├── config/              # Конфигурация
├── pkg/
│   ├── certstream/      # Клиент Certstream
│   ├── models/          # Модели данных
│   ├── phishing/        # Детектор фишинга
│   ├── storage/         # Работа с БД
│   ├── telegram/        # Telegram бот
│   └── updater/         # Обновление баз данных
├── data/                # База данных SQLite
├── docker-compose.yml   # Docker Compose
├── Dockerfile          # Docker образ
└── README.md           # Документация
```

### Локальная разработка

```bash
# Установка зависимостей
go mod download

# Запуск тестов
go test ./...

# Сборка
go build -o phishing-monitor cmd/monitor/main.go

# Форматирование кода
go fmt ./...
```

## 🤝 Вклад в проект

1. Форкните репозиторий
2. Создайте feature ветку (`git checkout -b feature/amazing-feature`)
3. Сделайте коммит (`git commit -m 'Add amazing feature'`)
4. Запушьте ветку (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📝 Лицензия

Этот проект распространяется под лицензией MIT. См. файл [LICENSE](LICENSE) для деталей.

## ⚠️ Ответственность

Этот инструмент предназначен только для легального мониторинга безопасности ваших собственных доменов или доменов клиентов с их явного согласия. Авторы не несут ответственности за неправомерное использование.

## 📞 Поддержка

Если у вас есть вопросы или проблемы:

1. Создайте [Issue](https://github.com/your-repo/phishing-monitor/issues)
2. Проверьте [FAQ](https://github.com/your-repo/phishing-monitor/wiki/FAQ)
3. Обратитесь в [Discussions](https://github.com/your-repo/phishing-monitor/discussions)

---

Сделано с ❤️ для SOC аналитиков
