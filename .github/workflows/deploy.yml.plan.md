# Docker Image Deployment Workflow Plan

## 概要
gobotプロジェクトのDocker Imageを自動的にビルド・デプロイするためのGitHub Actions Workflowの計画書

## 現状分析

### 既存リソース
1. **Dockerfile**: golang:1.24.4をベースにしたシングルステージビルド
2. **docker-compose.yml**: アプリケーションとMySQLデータベースの構成
3. **既存CI**: CI.yml (format, test, lint)、codeql.yml (セキュリティスキャン)
4. **ビルドツール**: GoReleaser設定あり (.goreleaser.yaml)

### プロジェクト情報
- **言語**: Go 1.24.4
- **依存関係**: MySQL, Redis
- **ビルド成果物**: gobotバイナリ
- **実行コマンド**: `./gobot bot --pprof --debug`

## デプロイ対象の特定

### Docker Imageタグ戦略

#### 1. タグ命名規則
```
ghcr.io/sabafly/gobot:latest          # mainブランチの最新
ghcr.io/sabafly/gobot:develop         # developブランチの最新
ghcr.io/sabafly/gobot:v1.2.3          # セマンティックバージョンタグ
ghcr.io/sabafly/gobot:sha-abc123      # 特定コミットSHA
ghcr.io/sabafly/gobot:pr-123          # PRプレビュービルド（オプション）
```

#### 2. ビルド・プッシュ条件

| トリガー | タグ | 説明 |
|---------|------|------|
| mainブランチへのpush | `latest`, `sha-{短縮SHA}` | 本番環境向け最新イメージ |
| セマンティックバージョンタグ (v*.*.*)  | `{version}`, `latest` | リリース版イメージ |
| developブランチへのpush | `develop`, `sha-{短縮SHA}` | 開発環境向けイメージ |
| Pull Request | なし（ビルドのみ実行） | イメージの検証のみ |

## レジストリ選択

### 推奨: GitHub Container Registry (GHCR)
**理由:**
- ✅ GitHubネイティブ統合（権限管理が簡単）
- ✅ パブリック/プライベートリポジトリの柔軟な管理
- ✅ 追加のサービス登録不要
- ✅ 無料枠が大きい（500MB/パッケージ、無制限ダウンロード）
- ✅ `GITHUB_TOKEN`で自動認証可能

### 代替案: Docker Hub
**メリット:**
- 広く認知されている
- DockerHubの検索可能性

**デメリット:**
- 別途アカウント管理が必要
- 無料プランの制限（6ヶ月未使用で削除）
- 追加のシークレット設定が必要

**決定**: GHCRを採用

## CI/CDフロー設計

### フェーズ1: 検証 (Validation)
```
┌─────────────────┐
│  PR作成/更新    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Code Quality   │
│  - format       │
│  - lint         │
│  - test         │
│  - codeql       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Docker Build    │
│ (push無し)      │
└─────────────────┘
```

### フェーズ2: ビルド・デプロイ (Build & Deploy)
```
┌─────────────────┐
│ main/develop    │
│ ブランチpush    │
│ またはタグ作成  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ CI Tests        │
│ (既存CI.yml)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Docker Build    │
│ - Multi-arch    │
│   (amd64,arm64) │
│ - Layer cache   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Security Scan   │
│ - Trivy         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Push to GHCR    │
│ - タグ付け      │
│ - メタデータ    │
└─────────────────┘
```

## ワークフロートリガー条件

### deploy.ymlトリガー設定
```yaml
on:
  push:
    branches:
      - main
      - develop
    tags:
      - 'v*.*.*'
  pull_request:
    branches:
      - main
      - develop
  workflow_dispatch:  # 手動実行を許可
```

### 説明
- **mainブランチpush**: 本番環境向けイメージ作成
- **developブランチpush**: 開発環境向けイメージ作成
- **タグpush (v*.*.*)**: リリース版イメージ作成
- **Pull Request**: ビルド検証のみ（プッシュなし）
- **workflow_dispatch**: 手動トリガー（緊急対応用）

## 必要なSecrets・権限

### GitHub Secrets（不要）
GHCRを使用する場合、`GITHUB_TOKEN`が自動的に利用可能なため、追加のシークレット設定は**不要**です。

### 必要な権限設定
```yaml
permissions:
  contents: read        # リポジトリコンテンツの読み取り
  packages: write       # GHCRへのプッシュ
  id-token: write       # OIDCトークン（オプション）
```

### Docker Hubを使用する場合（代替案）
以下のSecretsが必要:
- `DOCKERHUB_USERNAME`: Docker Hubユーザー名
- `DOCKERHUB_TOKEN`: Docker Hubアクセストークン

## deploy.yml 設計案

### ジョブ構成

#### Job 1: build-and-push
```yaml
name: Build and Deploy Docker Image

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    
    steps:
      - Checkout
      - Docker メタデータ生成
      - QEMU セットアップ（マルチアーチ用）
      - Docker Buildx セットアップ
      - GHCR ログイン
      - Docker イメージビルド＆プッシュ
      - Trivy セキュリティスキャン
      - ビルド成果物の出力
```

### マルチアーキテクチャサポート
```yaml
platforms: |
  linux/amd64
  linux/arm64
```

### キャッシュ戦略
```yaml
cache-from: type=gha
cache-to: type=gha,mode=max
```

GitHub Actionsキャッシュを利用してビルド時間を短縮

## セキュリティ対策

### 1. イメージスキャン
- **Trivy**: 脆弱性スキャン
  - CRITICALおよびHIGHの脆弱性を検出
  - スキャン結果をGitHub Security Advisoriesにアップロード

### 2. 署名（オプション）
- Sigstoreを使用したイメージ署名
- コンテナの完全性と出所の検証

### 3. 最小権限の原則
- 必要最小限のpermissionsを設定
- `GITHUB_TOKEN`の自動ローテーション

## Dockerfile最適化提案

### 現状のDockerfile問題点
1. シングルステージビルド（イメージサイズが大きい）
2. ビルド依存関係が含まれる
3. セキュリティアップデートが不明確

### 提案: マルチステージビルド
```dockerfile
# ビルドステージ
FROM golang:1.24.4 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gobot .

# 実行ステージ
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/gobot .
COPY --from=builder /app/lang ./lang
COPY --from=builder /app/locales ./locales
USER nonroot:nonroot
EXPOSE 6060
CMD ["./gobot", "bot", "--pprof", "--debug"]
```

**メリット:**
- イメージサイズの大幅削減（数百MB → 数十MB）
- セキュリティ向上（最小限のベースイメージ）
- 攻撃面の縮小

## 実装ステップ

### Phase 1: 基本実装
- [ ] deploy.ymlワークフローファイル作成
- [ ] GHCRへのpush設定
- [ ] 基本的なタグ戦略実装
- [ ] ドキュメント更新（README.md）

### Phase 2: 最適化
- [ ] Dockerfileのマルチステージビルド化
- [ ] マルチアーキテクチャビルド設定
- [ ] ビルドキャッシュ最適化

### Phase 3: セキュリティ強化
- [ ] Trivyスキャン統合
- [ ] セキュリティレポート自動作成
- [ ] イメージ署名（オプション）

### Phase 4: 運用改善
- [ ] 自動タグ付けの洗練
- [ ] ロールバック戦略
- [ ] モニタリング・通知設定

## モニタリング・通知

### 成功/失敗通知
- GitHub Actionsのステータスチェック
- （オプション）Slack/Discord通知統合

### メトリクス
- ビルド時間
- イメージサイズ
- プル回数（GHCR Insights）

## ロールバック戦略

### タグ管理でのロールバック
```bash
# 特定バージョンへのロールバック
docker pull ghcr.io/sabafly/gobot:v1.2.2

# 特定コミットへのロールバック
docker pull ghcr.io/sabafly/gobot:sha-abc123
```

### 推奨事項
- 過去のイメージを削除しない（少なくとも直近10バージョン保持）
- 重要なバージョンに明示的なタグを付ける

## 使用例

### ユーザー向けドキュメント（README.md追加案）

```markdown
## Docker を使用した実行

### GitHub Container Registry から最新イメージを取得

```bash
docker pull ghcr.io/sabafly/gobot:latest
```

### 特定バージョンを使用

```bash
docker pull ghcr.io/sabafly/gobot:v1.0.0
```

### docker-composeでの使用

```yaml
services:
  gobot:
    image: ghcr.io/sabafly/gobot:latest
    # ... その他の設定
```
```

## 参考リソース

### GitHub Actions
- [Publishing Docker images](https://docs.github.com/en/actions/publishing-packages/publishing-docker-images)
- [Working with the Container registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)

### Docker
- [Multi-platform images](https://docs.docker.com/build/building/multi-platform/)
- [Build cache](https://docs.docker.com/build/cache/)

### セキュリティ
- [Trivy](https://github.com/aquasecurity/trivy)
- [Sigstore](https://www.sigstore.dev/)

## 次のステップ

1. **このプラン文書のレビュー**
   - チームメンバーによる確認
   - フィードバックの収集

2. **実装Issueの作成**
   - Phase 1: 基本実装
   - Phase 2: 最適化
   - Phase 3: セキュリティ
   - Phase 4: 運用

3. **段階的な実装**
   - まずPR環境で検証
   - developブランチでテスト
   - mainブランチへのマージ

4. **ドキュメント整備**
   - README.mdの更新
   - CONTRIBUTING.mdの更新（あれば）

## まとめ

本プランでは、GitHub Container Registryを使用した自動Docker Imageデプロイワークフローを提案しました。

**主な特徴:**
- ✅ ゼロコンフィグ認証（GITHUB_TOKEN使用）
- ✅ マルチアーキテクチャサポート
- ✅ セキュリティスキャン統合
- ✅ 柔軟なタグ戦略
- ✅ 段階的な実装パス

このプランに基づいて、次の実装フェーズに進むことができます。
