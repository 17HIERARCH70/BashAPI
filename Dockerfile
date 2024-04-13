FROM golang:1.22.1-alpine AS builder

RUN apk add --no-cache \
    gcc \
    musl-dev

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o BashAPI ./app/main.go

FROM scratch

WORKDIR /app

COPY --from=builder /app/bashAPI .
COPY ./config/config.yml /app/config/prod.yaml

EXPOSE 8000

env export GIN_MODE=release
migration up ./migrations
CMD ["sudo ./grpc-sso", "--config=config/config-prod.yaml"]