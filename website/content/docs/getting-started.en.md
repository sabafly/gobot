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
- `/bbs` — Browse and post on Anonymous BBS boards and threads. Please review the [BBS Terms of Service](/en/bbs/terms/) and [BBS Guidelines](/en/bbs/guidelines/) before using the BBS.

See the [Commands](/en/docs/commands/) reference for full details on every public command and option.

## 3. Self-Hosting with Docker

Multi-architecture container images (`linux/amd64` and `linux/arm64`) are published to GitHub Container Registry:

```bash
docker pull ghcr.io/sabafly/gobot:latest
```

To run with the `docker-compose.yml` included in the repository:

```bash
cp .env-sample .env
# Edit .env with your Discord Bot Token and settings
docker compose up -d
```

For full source code and configuration details, visit the [GitHub repository](https://github.com/sabafly/gobot).
