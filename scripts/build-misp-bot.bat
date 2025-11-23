@echo off
chcp 65001 >nul
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║              MISP SOC Bot - Build Script                     ║
echo ╚══════════════════════════════════════════════════════════════╝
echo.

REM Переходим в корень проекта
cd /d "%~dp0.."

echo [BUILD] Сборка MISP SOC Bot для Windows...
echo.

REM Устанавливаем переменные окружения для Windows
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

REM Собираем приложение
echo [BUILD] Компиляция misp-bot.exe...
go build -ldflags="-s -w" -o misp-bot.exe ./cmd/misp-bot

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [ERROR] Ошибка сборки! Проверьте код.
    pause
    exit /b 1
)

echo.
echo [SUCCESS] Сборка завершена успешно!
echo [INFO] Файл: misp-bot.exe
echo.

REM Проверяем наличие .env файла
if not exist ".env" (
    echo [WARNING] Файл .env не найден!
    echo [INFO] Создайте файл .env на основе .env.example
    echo.
    if exist ".env.example" (
        echo [INFO] Копирую .env.example в .env...
        copy .env.example .env
        echo [INFO] Отредактируйте .env файл перед запуском!
    )
)

echo.
echo ========================================
echo   Для запуска выполните: misp-bot.exe
echo ========================================
echo.

pause
