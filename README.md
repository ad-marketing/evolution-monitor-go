# Evolution Monitor

Monitor automático de instâncias da [Evolution API](https://github.com/EvolutionAPI/evolution-api) v2.x com reconexão inteligente e notificações via **Telegram**.

Desenvolvido em **Go** — binário único, consumo mínimo de recursos (~5MB RAM), sem dependências externas.

## Como Funciona

1. A cada **1 minuto**, o monitor consulta todas as instâncias da Evolution API
2. Se uma instância estiver com status diferente de `open`, ele tenta **reconectar automaticamente**
3. Aguarda 10 segundos e verifica novamente. Repete até **3 tentativas**
4. Se após 3 tentativas a instância continuar offline, envia uma **notificação via Telegram** informando qual instância caiu
5. As configurações de Telegram e template de mensagem podem ser gerenciadas pelo **Dashboard web**

## Stack Completa

O projeto é composto por dois serviços:

| Serviço | Descrição | Imagem Docker |
|---------|-----------|---------------|
| **evolution-monitor** | Backend Go — monitoramento, reconexão, Telegram e API HTTP | `admarketing/evolution-monitor-go:latest` |
| **evolution-monitor-dashboard** | Frontend React — dashboard visual com tela de configurações | `admarketing/evolution-monitor-dashboard:latest` |

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

    volumes:
      - monitor_data:/data

    environment:
      - TZ=America/Sao_Paulo
      # ====== API MONITORADA ======
      - EVOLUTION_API_URL=https://SUA_URL_EVOLUTION_AQUI
      - EVOLUTION_API_KEY=SUA_API_KEY_AQUI
      # ====== INTERVALO E TENTATIVAS ======
      - CHECK_INTERVAL=60000
      - MAX_RESTART_ATTEMPTS=3
      - WAIT_AFTER_RESTART=10000
      # ====== TELEGRAM (pode ser configurado via dashboard) ======
      - TELEGRAM_BOT_TOKEN=
      - TELEGRAM_CHAT_ID=
      - TELEGRAM_ENABLED=true
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

volumes:
  monitor_data:

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
    volumes:
      - ./data:/data
    environment:
      - TZ=America/Sao_Paulo
      - EVOLUTION_API_URL=https://SUA_URL_EVOLUTION_AQUI
      - EVOLUTION_API_KEY=SUA_API_KEY_AQUI
      - CHECK_INTERVAL=60000
      - MAX_RESTART_ATTEMPTS=3
      - WAIT_AFTER_RESTART=10000
      - TELEGRAM_BOT_TOKEN=SEU_TOKEN_DO_BOT
      - TELEGRAM_CHAT_ID=SEU_CHAT_ID
      - TELEGRAM_ENABLED=true
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
| `TELEGRAM_BOT_TOKEN` | Token do bot do Telegram | - |
| `TELEGRAM_CHAT_ID` | Chat ID para receber notificações | - |
| `TELEGRAM_ENABLED` | Habilitar notificações via Telegram | `true` |
| `SERVER_PORT` | Porta do servidor HTTP | `3500` |
| `IGNORE_INSTANCES` | Instâncias a ignorar (separadas por vírgula) | - |
| `VERBOSE` | Logs detalhados para debug | `false` |

## Configuração do Telegram

### Como criar um Bot no Telegram

1. Abra o Telegram e busque por **@BotFather**
2. Envie o comando `/newbot`
3. Escolha um nome para o bot (ex: "Monitor Evolution")
4. Escolha um username (ex: `monitor_evolution_bot`)
5. O BotFather retornará o **Token** — copie e use em `TELEGRAM_BOT_TOKEN`

### Como obter o Chat ID

1. Abra o Telegram e busque por **@userinfobot**
2. Envie `/start` — ele retornará seu **Chat ID**
3. Ou: envie uma mensagem para seu bot, depois acesse:
   `https://api.telegram.org/bot<SEU_TOKEN>/getUpdates`
   O `chat.id` estará na resposta JSON

### Configuração via Dashboard

As configurações de Evolution API, Telegram e template de mensagem podem ser gerenciadas pela **tela de Configurações** do dashboard web (3 abas: Evolution | Telegram | Template), sem precisar reiniciar o container.

#### Aba Evolution
- URL da API
- API Key Global
- Intervalo de verificação (em segundos)

#### Aba Telegram
- Token do Bot
- Chat ID
- Ativar/Desativar notificações
- Tutorial integrado de como criar o bot

#### Aba Template
- Editor de mensagem personalizada
- Variáveis dinâmicas disponíveis: `{{instance_name}}`, `{{status}}`, `{{attempts}}`, `{{max_attempts}}`, `{{timestamp}}`, `{{server_url}}`
- Botão de teste para validar antes de salvar

## API HTTP (Endpoints)

O monitor expõe uma API REST na porta `3500` (configurável via `SERVER_PORT`):

| Endpoint | Método | Descrição |
|----------|--------|-----------|
| `/api/health` | GET | Health check do serviço |
| `/api/status` | GET | Resumo do último ciclo de monitoramento |
| `/api/instances` | GET | Lista detalhada de todas as instâncias e seus estados |
| `/api/stats` | GET | Estatísticas gerais (uptime, ciclos executados, etc.) |
| `/api/settings` | GET | Retorna configurações atuais (Evolution + Telegram + Template) |
| `/api/settings` | POST | Salva novas configurações (aceita parcial) |
| `/api/settings/test-notification` | POST | Envia notificação de teste |

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
curl http://localhost:3500/api/status
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
│   ├── server/
│   │   └── server.go         # Servidor HTTP (API para dashboard)
│   └── telegram/
│       └── telegram.go       # Cliente Telegram Bot API
├── Dockerfile                 # Build multi-stage (~25MB final)
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
| API HTTP | Não | Sim |
| Dashboard | Não | Sim (container separado) |
| Notificação | WhatsApp (Evolution) | Telegram |
| Configuração via UI | Não | Sim |
| Binário único | Não | Sim |

## Roadmap

- [x] Monitoramento e reconexão automática
- [x] Notificação via Telegram
- [x] Dashboard web (frontend React)
- [x] Configuração de Telegram via dashboard
- [x] Configuração da Evolution API via dashboard
- [x] Intervalo de verificação configurável via dashboard
- [x] Template de mensagem personalizável
- [x] Integração com Traefik + Docker Swarm
- [x] API HTTP para integração
- [ ] Histórico de eventos em banco de dados
- [ ] Webhook para integrações externas
- [ ] Autenticação no dashboard

---

*Desenvolvido por [Ad Marketing](https://github.com/ad-marketing)*
