#!/bin/bash

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              MISP SOC Bot - Build Script                     ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Переходим в корень проекта
cd "$(dirname "$0")/.."

echo "[BUILD] Сборка MISP SOC Bot..."
echo ""

# Сборка для текущей платформы
echo "[BUILD] Компиляция для текущей платформы..."
go build -ldflags="-s -w" -o misp-bot ./cmd/misp-bot

if [ $? -ne 0 ]; then
    echo ""
    echo "[ERROR] Ошибка сборки!"
    exit 1
fi

echo "[SUCCESS] Сборка для текущей платформы завершена: misp-bot"
echo ""

# Сборка для Windows
echo "[BUILD] Компиляция для Windows (misp-bot.exe)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o misp-bot.exe ./cmd/misp-bot

if [ $? -ne 0 ]; then
    echo ""
    echo "[ERROR] Ошибка сборки для Windows!"
    exit 1
fi

echo "[SUCCESS] Сборка для Windows завершена: misp-bot.exe"
echo ""

# Проверяем наличие .env файла
if [ ! -f ".env" ]; then
    echo "[WARNING] Файл .env не найден!"
    echo "[INFO] Создайте файл .env на основе .env.example"
    if [ -f ".env.example" ]; then
        echo "[INFO] Копирую .env.example в .env..."
        cp .env.example .env
        echo "[INFO] Отредактируйте .env файл перед запуском!"
    fi
fi

echo ""
echo "========================================"
echo "  Для запуска выполните:"
echo "  Linux/Mac: ./misp-bot"
echo "  Windows:   misp-bot.exe"
echo "========================================"
echo ""
