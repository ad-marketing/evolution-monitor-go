# Evolution Monitor (Go)

Monitor automático de instâncias da Evolution API com reconexão automática, notificação via WhatsApp e API HTTP para dashboard.

Escrito em **Go** — binário único, consumo mínimo de recursos (~5MB RAM), sem dependências externas.

## Funcionalidades

- **Monitoramento Contínuo**: Verifica o status de todas as instâncias a cada 1 minuto (configurável)
- **Auto-Reconexão**: Tenta restart automático até 3 vezes quando uma instância desconecta
- **Notificação via WhatsApp**: Envia alerta detalhado quando a reconexão falha
- **API de Notificação Externa**: Permite enviar alertas por outra Evolution API (recomendado)
- **API HTTP**: Endpoints REST prontos para integração com dashboard/frontend
- **Integração com Traefik**: Labels prontas para SSL automático via Let's Encrypt
- **Ultra-leve**: Imagem Docker de ~10MB, consumo de ~5MB de RAM

## Instalação via Portainer (Docker Swarm + Traefik)

1. Vá em **Stacks** → **Add Stack**
2. Dê o nome `evolution-monitor-go`
3. Cole o conteúdo do `docker-compose.yml` no editor
4. Preencha as variáveis com seus dados
5. Clique em **Deploy the stack**

## Instalação via Terminal (Docker Swarm)

```bash
mkdir -p /opt/evolution-monitor-go && cd /opt/evolution-monitor-go
```

Crie o `docker-compose.yml`:

```yaml
version: "3.7"
services:
  evolution-monitor:
    image: admarketing/evolution-monitor-go:latest

    networks:
      - SuaRedeAqui ## Nome da rede interna (mesma do Traefik)

    environment:
      - TZ=America/Sao_Paulo
      # ====== API MONITORADA ======
      - EVOLUTION_API_URL=https://SUA_URL_EVOLUTION_AQUI
      - EVOLUTION_API_KEY=SUA_API_KEY_AQUI
      # ====== INTERVALO E TENTATIVAS ======
      - CHECK_INTERVAL=60000
      - MAX_RESTART_ATTEMPTS=3
      - WAIT_AFTER_RESTART=10000
      # ====== NOTIFICAÇÃO VIA API EXTERNA ======
      - NOTIFICATION_API_URL=https://SUA_URL_EVOLUTION_AQUI
      - NOTIFICATION_API_KEY=SUA_API_KEY_AQUI
      - NOTIFICATION_SENDER_INSTANCE=INSTANCIA_NAME_AQUI
      - NOTIFICATION_ADMIN_NUMBER=NUMERO_WHATSAPP_AQUI
      - NOTIFICATION_ENABLED=true
      # ====== SERVIDOR HTTP (DASHBOARD) ======
      - SERVER_PORT=3500
      # ====== CONFIGURAÇÕES AVANÇADAS ======
      - IGNORE_INSTANCES=
      - VERBOSE=false

    deploy:
      mode: replicated
      replicas: 1
      placement:
        constraints:
          - node.role == manager
      labels:
        - traefik.enable=1
        - traefik.http.routers.evolution-monitor.rule=Host(`monitor.seudominio.com.br`) ## URL do Dashboard
        - traefik.http.routers.evolution-monitor.entrypoints=websecure
        - traefik.http.routers.evolution-monitor.priority=1
        - traefik.http.routers.evolution-monitor.tls.certresolver=letsencryptresolver
        - traefik.http.routers.evolution-monitor.service=evolution-monitor
        - traefik.http.services.evolution-monitor.loadbalancer.server.port=3500
        - traefik.http.services.evolution-monitor.loadbalancer.passHostHeader=true

    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

networks:
  SuaRedeAqui: ## Nome da rede interna (mesma do Traefik)
    external: true
    name: SuaRedeAqui ## Nome da rede interna (mesma do Traefik)
```

Faça o deploy via stack:

```bash
docker stack deploy -c docker-compose.yml evolution-monitor
```

## Instalação Simples (sem Traefik)

Se não usa Traefik/Swarm, pode rodar com `network_mode: host`:

```yaml
services:
  evolution-monitor:
    image: admarketing/evolution-monitor-go:latest
    container_name: evolution-monitor-go
    restart: unless-stopped
    environment:
      - TZ=America/Sao_Paulo
      - EVOLUTION_API_URL=https://SUA_URL_EVOLUTION_AQUI
      - EVOLUTION_API_KEY=SUA_API_KEY_AQUI
      - CHECK_INTERVAL=60000
      - MAX_RESTART_ATTEMPTS=3
      - WAIT_AFTER_RESTART=10000
      - NOTIFICATION_API_URL=https://SUA_URL_EVOLUTION_AQUI
      - NOTIFICATION_API_KEY=SUA_API_KEY_AQUI
      - NOTIFICATION_SENDER_INSTANCE=INSTANCIA_NAME_AQUI
      - NOTIFICATION_ADMIN_NUMBER=NUMERO_WHATSAPP_AQUI
      - NOTIFICATION_ENABLED=true
      - SERVER_PORT=3500
      - IGNORE_INSTANCES=
      - VERBOSE=false
    network_mode: host
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

```bash
docker compose up -d
```

## API HTTP (para Dashboard)

O monitor expõe uma API REST na porta `3500` (configurável via `SERVER_PORT`):

| Endpoint | Método | Descrição |
|----------|--------|-----------|
| `/api/health` | GET | Health check do serviço |
| `/api/status` | GET | Resumo do último ciclo de monitoramento |
| `/api/instances` | GET | Lista detalhada de todas as instâncias e seus estados |
| `/api/stats` | GET | Estatísticas gerais (uptime, ciclos executados, etc.) |

### Exemplos de resposta

**GET /api/status**
```json
{
  "status": "active",
  "last_check": "2026-05-29T15:30:00-03:00",
  "total": 3,
  "ok": 2,
  "reconnected": 1,
  "failed": 0,
  "ignored": 0
}
```

**GET /api/instances**
```json
[
  {
    "name": "MinhaInstancia",
    "state": "open",
    "status": "ok",
    "last_check": "2026-05-29T15:30:00-03:00"
  }
]
```

## Configurações

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `EVOLUTION_API_URL` | URL da Evolution API monitorada | - |
| `EVOLUTION_API_KEY` | API Key global da API monitorada | - |
| `CHECK_INTERVAL` | Intervalo de verificação em ms | `60000` (1 min) |
| `MAX_RESTART_ATTEMPTS` | Tentativas de restart antes de notificar | `3` |
| `WAIT_AFTER_RESTART` | Espera após restart (ms) | `10000` (10s) |
| `NOTIFICATION_API_URL` | URL da API que ENVIA as notificações | Mesma da monitorada |
| `NOTIFICATION_API_KEY` | API Key da API de notificação | Mesma da monitorada |
| `NOTIFICATION_SENDER_INSTANCE` | Instância que envia os alertas | - |
| `NOTIFICATION_ADMIN_NUMBER` | Número WhatsApp do admin (DDI+DDD+Número) | - |
| `NOTIFICATION_ENABLED` | Habilitar notificações | `true` |
| `SERVER_PORT` | Porta do servidor HTTP | `3500` |
| `IGNORE_INSTANCES` | Instâncias a ignorar (separadas por vírgula) | - |
| `VERBOSE` | Logs detalhados para debug | `false` |

## Comparativo com versão Node.js

| Métrica | Node.js | Go |
|---------|---------|-----|
| Imagem Docker | ~50MB | ~10MB |
| RAM em uso | ~30-50MB | ~5MB |
| Dependências | dotenv | Nenhuma |
| API HTTP | Não | Sim (pronto para dashboard) |
| Binário único | Não | Sim |
| Integração Traefik | Manual | Labels prontas |

## Comandos Úteis

```bash
# Ver logs em tempo real (Swarm)
docker service logs evolution-monitor_evolution-monitor -f

# Ver logs (compose)
docker logs evolution-monitor-go -f

# Atualizar para última versão (Swarm)
docker service update --image admarketing/evolution-monitor-go:latest evolution-monitor_evolution-monitor

# Atualizar (compose)
docker compose pull && docker compose up -d

# Testar API
curl https://monitor.seudominio.com.br/api/status
```

## Estrutura do Projeto

```
.
├── main.go                    # Ponto de entrada
├── internal/
│   ├── api/
│   │   └── client.go         # Cliente HTTP para Evolution API
│   ├── config/
│   │   └── config.go         # Carregamento de configurações
│   ├── monitor/
│   │   └── monitor.go        # Lógica de monitoramento e reconexão
│   └── server/
│       └── server.go         # Servidor HTTP (API para dashboard)
├── Dockerfile                 # Build multi-stage (~10MB final)
├── docker-compose.yml         # Compose para deploy (Swarm + Traefik)
├── go.mod                     # Módulo Go
└── README.md
```

## Roadmap

- [x] Monitoramento e reconexão automática
- [x] Notificação via WhatsApp
- [x] API de notificação externa
- [x] API HTTP para integração
- [x] Integração com Traefik + Docker Swarm
- [ ] Dashboard web (frontend)
- [ ] Histórico de eventos em banco de dados
- [ ] Webhook para integrações externas

---

*Desenvolvido por [Ad Marketing](https://github.com/ad-marketing)*
