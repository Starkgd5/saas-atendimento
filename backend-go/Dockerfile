# Estágio 1: Build
FROM golang:1.22.2-alpine AS builder

WORKDIR /app

# Baixar dependências
COPY go.mod go.sum ./
RUN go mod download

# Copiar código e compilar
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api ./cmd/api

# Estágio 2: Final (imagem minúscula ~15MB)
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copiar binário e arquivos de configuração
COPY --from=builder /app/api .
COPY --from=builder /app/.env* ./

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["./api", "-health"]

CMD ["./api"]