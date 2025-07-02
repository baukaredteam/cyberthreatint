#!/bin/bash

# Скрипт помощи в настройке Telegram бота

echo "🤖 Помощник настройки Telegram бота"
echo ""

# Функция для получения Chat ID
get_chat_id() {
    if [ -z "$1" ]; then
        echo "❌ Необходимо указать токен бота"
        return 1
    fi
    
    local token=$1
    echo "📡 Получение обновлений для бота..."
    
    local response=$(curl -s "https://api.telegram.org/bot$token/getUpdates")
    
    if echo "$response" | grep -q '"ok":true'; then
        echo "✅ Бот работает корректно"
        
        # Извлекаем Chat ID из ответа
        local chat_ids=$(echo "$response" | grep -o '"chat":{"id":[^,]*' | grep -o '[0-9-]\+' | sort -u)
        
        if [ -n "$chat_ids" ]; then
            echo ""
            echo "📋 Найденные Chat ID:"
            for id in $chat_ids; do
                echo "   $id"
            done
            echo ""
            echo "💡 Используйте один из этих ID в переменной TELEGRAM_CHAT_ID"
        else
            echo ""
            echo "⚠️  Chat ID не найдены. Попробуйте:"
            echo "   1. Отправить боту сообщение /start"
            echo "   2. Добавить бота в групповой чат"
            echo "   3. Запустить этот скрипт снова"
        fi
    else
        echo "❌ Ошибка: $response"
        echo ""
        echo "Возможные причины:"
        echo "   1. Неверный токен бота"
        echo "   2. Бот заблокирован"
        echo "   3. Проблемы с сетью"
    fi
}

# Проверка аргументов
if [ "$1" = "get-chat-id" ] && [ -n "$2" ]; then
    get_chat_id "$2"
    exit 0
fi

echo "Инструкция по настройке Telegram бота:"
echo ""
echo "1️⃣  Создание бота:"
echo "   • Найдите @BotFather в Telegram"
echo "   • Отправьте команду /newbot"
echo "   • Следуйте инструкциям для создания бота"
echo "   • Скопируйте токен бота"
echo ""
echo "2️⃣  Получение Chat ID:"
echo "   • Отправьте боту сообщение /start"
echo "   • Запустите: ./scripts/telegram-setup.sh get-chat-id YOUR_BOT_TOKEN"
echo ""
echo "3️⃣  Настройка переменных окружения:"
echo "   • Откройте файл .env"
echo "   • Установите TELEGRAM_BOT_TOKEN=ваш_токен"
echo "   • Установите TELEGRAM_CHAT_ID=ваш_chat_id"
echo ""
echo "4️⃣  Проверка настройки:"
echo "   • Запустите сервис"
echo "   • Отправьте боту команду /status"
echo ""

# Проверка наличия .env файла
if [ -f .env ]; then
    # Проверяем настройки в .env
    if grep -q "TELEGRAM_BOT_TOKEN=YOUR_BOT_TOKEN_HERE" .env; then
        echo "⚠️  Не забудьте обновить TELEGRAM_BOT_TOKEN в файле .env"
    fi
    
    if grep -q "TELEGRAM_CHAT_ID=YOUR_CHAT_ID_HERE" .env; then
        echo "⚠️  Не забудьте обновить TELEGRAM_CHAT_ID в файле .env"
    fi
else
    echo "❌ Файл .env не найден. Запустите сначала scripts/setup.sh"
fi

echo ""
echo "Примеры использования:"
echo "   ./scripts/telegram-setup.sh get-chat-id 1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
echo ""