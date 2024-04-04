# Этап 1: Сборка Go-приложения
FROM golang:1.21 AS builder

WORKDIR /app

# Копируем исходный код в образ
COPY . .
# Скачиваем зависимости и собираем бинарный файл
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bashapi .

# Этап 2: Создаем конечный образ
FROM alpine:latest

WORKDIR /root/

# Копируем бинарный файл из предыдущего этапа
COPY --from=builder /app/bashapi .

# Копируем миграции
COPY --from=builder /app/migrations /migrations

# Устанавливаем migrate tool
RUN apk add --no-cache curl \
    && curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz \
    && mv migrate.linux-amd64 /usr/local/bin/migrate \
    && chmod +x /usr/local/bin/migrate \

RUN apk add --no-cache curl jq bash \
    && curl -L https://github.com/mikefarah/yq/releases/download/v4.6.1/yq_linux_amd64 -o /usr/bin/yq \
    && chmod +x /usr/bin/yq
# Скрипт для запуска миграций и приложения
COPY config/config.yml .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

ENTRYPOINT ["/root/entrypoint.sh"]
