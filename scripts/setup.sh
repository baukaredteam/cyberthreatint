#!/bin/bash

# Скрипт быстрой настройки Phishing Monitor

echo "🛡️  Настройка Phishing Monitor..."

# Проверка наличия Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен. Установите Go 1.21+ и попробуйте снова."
    exit 1
fi

# Проверка версии Go
GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
if [[ "$(printf '%s\n' "1.21" "$GO_VERSION" | sort -V | head -n1)" != "1.21" ]]; then
    echo "❌ Требуется Go версии 1.21 или выше. Текущая версия: $GO_VERSION"
    exit 1
fi

echo "✅ Go $GO_VERSION обнаружен"

# Создание директорий
echo "📁 Создание необходимых директорий..."
mkdir -p data logs

# Копирование файла конфигурации
if [ ! -f .env ]; then
    cp .env.example .env
    echo "📋 Создан файл .env из шаблона"
    echo "⚠️  ВАЖНО: Отредактируйте .env файл с вашими настройками!"
    echo "   - TELEGRAM_BOT_TOKEN: Токен вашего Telegram бота"
    echo "   - TELEGRAM_CHAT_ID: ID чата для уведомлений"
    echo "   - CLIENT_DOMAINS: Домены ваших клиентов"
else
    echo "✅ Файл .env уже существует"
fi

# Установка зависимостей
echo "📦 Установка зависимостей Go..."
go mod download

# Проверка компиляции
echo "🔨 Проверка компиляции..."
if go build -o /tmp/phishing-monitor-test ./cmd/monitor/main.go; then
    echo "✅ Компиляция успешна"
    rm -f /tmp/phishing-monitor-test
else
    echo "❌ Ошибка компиляции"
    exit 1
fi

echo ""
echo "🎉 Настройка завершена!"
echo ""
echo "Следующие шаги:"
echo "1. Отредактируйте файл .env с вашими настройками"
echo "2. Создайте Telegram бота через @BotFather"
echo "3. Получите Chat ID для уведомлений"
echo "4. Запустите сервис: go run cmd/monitor/main.go"
echo ""
echo "Для Docker развертывания:"
echo "1. Настройте .env файл"
echo "2. Запустите: docker-compose up -d"
echo ""
echo "📚 Подробная документация в README.md"