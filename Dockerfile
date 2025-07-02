# Многоэтапная сборка для оптимизации размера
FROM golang:1.21-alpine AS builder

# Установка зависимостей
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Создание рабочей директории
WORKDIR /app

# Копирование файлов зависимостей
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o phishing-monitor ./cmd/monitor/main.go

# Финальный образ
FROM alpine:latest

# Установка необходимых пакетов
RUN apk --no-cache add ca-certificates sqlite

# Создание рабочей директории
WORKDIR /root/

# Копирование бинарника
COPY --from=builder /app/phishing-monitor .

# Создание директории для данных
RUN mkdir -p /root/data

# Порт для healthcheck (опционально)
EXPOSE 8080

# Запуск приложения
CMD ["./phishing-monitor"]