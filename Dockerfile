# ============================================
# Build Stage
# ============================================
FROM golang:1.26.4 AS builder

WORKDIR /app

# 依存関係のキャッシュを最適化
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# generate が必要な場合は実行
# RUN go generate ./...

# バイナリをビルド
# CGO_ENABLED=0: 静的リンクでビルド（distrolessイメージで実行可能）
# -ldflags="-s -w": デバッグ情報を削除してバイナリサイズを削減
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o gobot \
    .

# ============================================
# Runtime Stage (Distroless)
# ============================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

# ビルドステージからバイナリをコピー
COPY --from=builder /app/gobot .

# ランタイムに必要なファイルをコピー
COPY --from=builder /app/lang ./lang
COPY --from=builder /app/locales ./locales

# 非rootユーザーで実行（セキュリティベストプラクティス）
USER nonroot:nonroot

# pprofポートを公開
EXPOSE 6060

# エントリーポイント
ENTRYPOINT ["./gobot"]
CMD ["bot", "--pprof", "--debug"]
