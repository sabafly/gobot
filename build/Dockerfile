# goバージョン
FROM golang:1.19.3-alpine
# アップデートとgitのインストール
RUN apk update && apk add git
# gobotディレクトリの作成
RUN mkdir /go/src/gobot/
# ワーキングディレクトリの設定
WORKDIR /go/src/gobot/
# ホストのファイルをコンテナの作業ディレクトリに移行
ADD . /go/src/gobot/
# パッケージのインポート
RUN go get -u golang.org/x/tools/cmd/goimports