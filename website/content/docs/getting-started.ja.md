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
- `/bbs` — 匿名掲示板（BBS）の板やスレッドを利用します。BBS の利用にあたっては [BBS 利用規約](/bbs/terms/) および [BBS ガイドライン](/bbs/guidelines/) をご確認ください。

各コマンドの詳細なオプションやサブコマンドは [コマンド一覧](/docs/commands/) を参照してください。

## 3. セルフホスト（Docker）

gobot は GitHub Container Registry（GHCR）でマルチアーキテクチャイメージ（`linux/amd64` / `linux/arm64`）を公開しています。

```bash
docker pull ghcr.io/sabafly/gobot:latest
```

リポジトリ内の `docker-compose.yml` を使って起動する場合は、`.env-sample` をコピーして環境変数を設定します。

```bash
cp .env-sample .env
# .env を編集して Discord Bot Token などを設定
docker compose up -d
```

ソースコードと最新の構成方法は [GitHub リポジトリ](https://github.com/sabafly/gobot) を参照してください。
