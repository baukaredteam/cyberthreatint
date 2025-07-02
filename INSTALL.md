# 🚀 Быстрая установка Phishing Monitor

## Предварительные требования

- Go 1.21 или выше
- Git
- Telegram Bot Token

## Быстрый старт

### 1. Клонирование и настройка

```bash
git clone <repository-url>
cd phishing-monitor
chmod +x scripts/*.sh
./scripts/setup.sh
```

### 2. Настройка Telegram бота

```bash
# Получите токен от @BotFather в Telegram
./scripts/telegram-setup.sh
```

### 3. Конфигурация

Отредактируйте файл `.env`:

```bash
nano .env
```

Минимальные настройки:
```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_CHAT_ID=your_chat_id_here
CLIENT_DOMAINS=qazpost.kz,yourcompany.com
```

### 4. Запуск

```bash
go run cmd/monitor/main.go
```

## Docker (рекомендуется)

```bash
# Настройте .env файл
cp .env.example .env
nano .env

# Запуск
docker-compose up -d

# Просмотр логов
docker-compose logs -f
```

## Проверка работы

1. Отправьте боту команду `/start`
2. Проверьте статус командой `/status`
3. Просмотрите логи приложения

## Поддержка

- 📖 [Полная документация](README.md)
- 🐛 [Сообщить об ошибке](issues)
- 💬 [Обсуждения](discussions)