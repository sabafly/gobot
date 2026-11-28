---
title: 導入ガイド
weight: 1
---

gobot をあなたの Discord サーバーに導入して使い始めるまでの手順です。

## 1. サーバーへ招待する

以下の招待リンクから、管理権限のある Discord サーバーに gobot を追加できます。

- [gobot をサーバーに招待する](https://discord.com/api/oauth2/authorize?client_id=1083042729996603412&permissions=8&scope=bot%20applications.commands)

サポートや最新の更新情報については、[公式 Discord サーバー](https://discord.gg/hnNBD8QNmB)をご利用ください。

## 2. 初期設定を行う

招待後、サーバー内で以下のスラッシュコマンドから必要な機能を有効化できます。

- `/setting` — サーバーごとの基本設定や通知チャンネルなどを構成します。
- `/permission` — コマンドの実行権限やロール制御を調整します。
- `/bbs` — 匿名掲示板（BBS）の板やスレッドを利用します。詳しい使い方は [匿名掲示板 (BBS) ガイド](/docs/bbs/)、規約は [BBS 利用規約](/bbs/terms/) および [BBS ガイドライン](/bbs/guidelines/) をご確認ください。

各コマンドの詳細なオプションやサブコマンドは [コマンド一覧](/docs/commands/) を参照してください。

## 3. セルフホスト（Docker）

gobot は GitHub Container Registry（GHCR）でマルチアーキテクチャイメージ（`linux/amd64` / `linux/arm64`）を公開しています。

```bash
docker pull ghcr.io/sabafly/gobot:latest
```

リポジトリ内の `docker-compose.yml` を使って起動する場合は、ルートの `.env` に加えて、MariaDB コンテナおよび gobot コンテナが `env_file` として読み込む `mysql/.env` を作成する必要があります。

```bash
# 1. Bot 用の環境変数ファイルを作成
cp .env-sample .env
# .env を編集して Discord Bot Token などを設定

# 2. MariaDB 用の環境変数ファイルを作成（docker-compose.yml で必須）
cat <<'EOF' > mysql/.env
MYSQL_ROOT_PASSWORD=change_me_root_password
MYSQL_DATABASE=gobot
MYSQL_USER=gobot
MYSQL_PASSWORD=change_me_db_password
EOF

# 3. コンテナを起動
docker compose up -d
```

- `MYSQL_ROOT_PASSWORD` は `mysql/Dockerfile` のヘルスチェック（`mariadb-admin ping`）でも使用されるため、必ず設定してください。
- ソースコードと最新の構成方法は [GitHub リポジトリ](https://github.com/sabafly/gobot) を参照してください。
