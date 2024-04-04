#!/bin/bash

# Чтение конфигурации Postgres
PG_HOST=$(yq e '.postgres.host' config.yml)
PG_PORT=$(yq e '.postgres.port' config.yml)
PG_USER=$(yq e '.postgres.user' config.yml)
PG_PASSWORD=$(yq e '.postgres.password' config.yml)
PG_DB=$(yq e '.postgres.database' config.yml)
PG_SSL_MODE=$(yq e '.postgres.ssl_mode' config.yml)

# Формирование строки подключения
DATABASE_URL="postgres://${PG_USER}:${PG_PASSWORD}@${PG_HOST}:${PG_PORT}/${PG_DB}?sslmode=${PG_SSL_MODE}"

# Экспорт переменной окружения DATABASE_URL
export DATABASE_URL

# Выполнение миграций
migrate -path /migrations -database "$DATABASE_URL" up

# Запуск приложения
./bashapi -config="config.yml"
