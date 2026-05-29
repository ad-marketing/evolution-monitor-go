# Evolution Monitor

Monitor automático de instâncias da [Evolution API](https://github.com/EvolutionAPI/evolution-api) v2.x com reconexão inteligente e notificações via WhatsApp.

Desenvolvido em **Go** — binário único, consumo mínimo de recursos (~5MB RAM), sem dependências externas.

## Como Funciona

1. A cada **1 minuto**, o monitor consulta todas as instâncias da Evolution API
2. Se uma instância estiver com status diferente de `open`, ele tenta **reconectar automaticamente**
3. Aguarda 10 segundos e verifica novamente. Repete até **3 tentativas**
4. Se após 3 tentativas a instância continuar offline, envia uma **notificação no WhatsApp** informando qual instância caiu
5. A notificação pode ser enviada por uma **API externa** (recomendado), garantindo que o alerta chegue mesmo se a API monitorada estiver completamente fora

## Stack Completa

O projeto é composto por dois serviços:

| Serviço | Descrição | Imagem Docker |
|---------|-----------|---------------|
| **evolution-monitor** | Backend Go — monitoramento, reconexão e API HTTP | `admarketing/evolution-monitor-go:latest` |
| **evolution-monitor-dashboard** | Frontend React — dashboard visual em tempo real | `admarketing/evolution-monitor-dashboard:latest` |

## Instalação via Portainer (Docker Swarm + Traefik)

1. Vá em **Stacks** → **Add Stack**
2. Dê o nome `evolution-monitor`
3. Cole o conteúdo do `docker-compose.yml` no editor
4. Substitua os valores das variáveis com seus dados
5. Clique em **Deploy the stack**

## Docker Compose (Stack Completa)

```yaml
version: "3.7"

services:
  ## ====== MONITOR (Backend Go) ======
  evolution-monitor:
    image: admarketing/evolution-monitor-go:latest

    networks:
      - SuaRedeAqui

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
      # Pode ser a mesma API monitorada ou outra API externa para enviar alertas
      - NOTIFICATION_API_URL=https://SUA_URL_EVOLUTION_AQUI
      - NOTIFICATION_API_KEY=SUA_API_KEY_AQUI
      - NOTIFICATION_SENDER_INSTANCE=INSTANCIA_QUE_ENVIA_ALERTA
      - NOTIFICATION_ADMIN_NUMBER=5500000000000
      - NOTIFICATION_ENABLED=true
      # ====== SERVIDOR HTTP (DASHBOARD API) ======
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

    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  ## ====== DASHBOARD (Frontend React) ======
  evolution-monitor-dashboard:
    image: admarketing/evolution-monitor-dashboard:latest

    networks:
      - SuaRedeAqui

    deploy:
      mode: replicated
      replicas: 1
      placement:
        constraints:
          - node.role == manager
      labels:
        - traefik.enable=true
        - traefik.http.routers.evo-monitor-dashboard.rule=Host(`monitor.seudominio.com.br`)
        - traefik.http.routers.evo-monitor-dashboard.entrypoints=websecure
        - traefik.http.routers.evo-monitor-dashboard.priority=1
        - traefik.http.routers.evo-monitor-dashboard.tls.certresolver=letsencryptresolver
        - traefik.http.routers.evo-monitor-dashboard.service=evo-monitor-dashboard
        - traefik.http.services.evo-monitor-dashboard.loadbalancer.server.port=80
        - traefik.http.services.evo-monitor-dashboard.loadbalancer.passHostHeader=true

    logging:
      driver: "json-file"
      options:
        max-size: "5m"
        max-file: "2"

networks:
  SuaRedeAqui:
    external: true
    name: SuaRedeAqui
```

## Instalação Simples (sem Traefik/Swarm)

Se não usa Traefik/Swarm, pode rodar com `network_mode: host`:

```yaml
services:
  evolution-monitor:
    image: admarketing/evolution-monitor-go:latest
    container_name: evolution-monitor
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
      - NOTIFICATION_SENDER_INSTANCE=INSTANCIA_QUE_ENVIA_ALERTA
      - NOTIFICATION_ADMIN_NUMBER=5500000000000
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

## Variáveis de Ambiente

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

## API HTTP (Endpoints do Dashboard)

O monitor expõe uma API REST na porta `3500` (configurável via `SERVER_PORT`):

| Endpoint | Método | Descrição |
|----------|--------|-----------|
| `/api/health` | GET | Health check do serviço |
| `/api/status` | GET | Resumo do último ciclo de monitoramento |
| `/api/instances` | GET | Lista detalhada de todas as instâncias e seus estados |
| `/api/stats` | GET | Estatísticas gerais (uptime, ciclos executados, etc.) |

### Exemplos de Resposta

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

## Notificação via API Externa

Se você monitora uma VPS que possui apenas **uma instância**, recomendamos configurar a notificação via uma **API externa** (outra VPS). Assim, mesmo que a instância monitorada caia, o alerta será enviado por outro caminho.

**Exemplo:**
- VPS A (monitorada): tem a instância `PDMReservas`
- VPS B (notificação): tem a instância `AdMarketingAPI`

Configure:
```
EVOLUTION_API_URL=https://evo.vps-a.com.br       # API monitorada
NOTIFICATION_API_URL=https://evo.vps-b.com.br    # API que envia o alerta
NOTIFICATION_SENDER_INSTANCE=AdMarketingAPI       # Instância da VPS B
```

## Comandos Úteis

```bash
# Ver logs em tempo real (Swarm)
docker service logs evolution-monitor_evolution-monitor -f

# Ver logs (compose)
docker logs evolution-monitor -f

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

## Comparativo com Versão Node.js

| Métrica | Node.js | Go |
|---------|---------|-----|
| Imagem Docker | ~50MB | ~25MB |
| RAM em uso | ~30-50MB | ~5MB |
| Dependências | dotenv | Nenhuma |
| API HTTP | Não | Sim (pronto para dashboard) |
| Binário único | Não | Sim |
| Integração Traefik | Manual | Labels prontas |

## Roadmap

- [x] Monitoramento e reconexão automática
- [x] Notificação via WhatsApp
- [x] API de notificação externa
- [x] API HTTP para integração
- [x] Integração com Traefik + Docker Swarm
- [x] Dashboard web (frontend React)
- [ ] Histórico de eventos em banco de dados
- [ ] Webhook para integrações externas
- [ ] Autenticação no dashboard

---

*Desenvolvido por [Ad Marketing](https://github.com/ad-marketing)*
