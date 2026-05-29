# ============================================================
# Stage 1: Build
# ============================================================
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiar go.mod
COPY go.mod ./

# Copiar código fonte
COPY . .

# Build do binário estático
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /monitor .

# ============================================================
# Stage 2: Runtime (imagem mínima ~10MB)
# ============================================================
FROM alpine:3.19

# Timezone e certificados SSL
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copiar binário do stage de build
COPY --from=builder /monitor .

# Criar usuário não-root
RUN addgroup -g 1001 -S monitor && \
    adduser -S monitor -u 1001 -G monitor

USER monitor

# Porta do servidor HTTP (para frontend futuro)
EXPOSE 3500

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:3500/api/health || exit 1

ENTRYPOINT ["./monitor"]
