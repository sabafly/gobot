---
title: Getting Started
weight: 1
---

Follow these steps to add gobot to your Discord server and configure its core features.

## 1. Invite gobot to Your Server

Use the authorization link below to add gobot to a Discord server where you have Manage Server permissions:

- [Invite gobot to Discord](https://discord.com/api/oauth2/authorize?client_id=1083042729996603412&permissions=8&scope=bot%20applications.commands)

Need help or want to follow updates? Join the [official Discord server](https://discord.gg/hnNBD8QNmB).

## 2. Initial Setup

Once gobot joins your server, you can configure features with slash commands:

- `/setting` — Configure server-level settings and channels.
- `/permission` — Manage command permissions and role restrictions.
- `/bbs` — Browse and post on Anonymous BBS boards and threads. See the [Anonymous BBS Guide](/en/docs/bbs/) for setup and usage, and review the [BBS Terms of Service](/en/bbs/terms/) and [BBS Guidelines](/en/bbs/guidelines/) before using the BBS.

See the [Commands](/en/docs/commands/) reference for full details on every public command and option.

## 3. Self-Hosting with Docker

Multi-architecture container images (`linux/amd64` and `linux/arm64`) are published to GitHub Container Registry:

```bash
docker pull ghcr.io/sabafly/gobot:latest
```

To run with the `docker-compose.yml` included in the repository, you must create both `.env` at the repository root and `mysql/.env`, which `docker-compose.yml` loads via `env_file` for the `go`, `mysql` (MariaDB), and `sql-gui` services:

```bash
# 1. Create the bot environment file
cp .env-sample .env
# Edit .env with your Discord Bot Token and settings

# 2. Create the MariaDB environment file required by docker-compose.yml
cat <<'EOF' > mysql/.env
MYSQL_ROOT_PASSWORD=change_me_root_password
MYSQL_DATABASE=gobot
MYSQL_USER=gobot
MYSQL_PASSWORD=change_me_db_password
EOF

# 3. Start the containers
docker compose up -d
```

- `MYSQL_ROOT_PASSWORD` is also required by the MariaDB healthcheck in `mysql/Dockerfile` (`mariadb-admin ping -h localhost -p$MYSQL_ROOT_PASSWORD`).
- For full source code and configuration details, visit the [GitHub repository](https://github.com/sabafly/gobot).
