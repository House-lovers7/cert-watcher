# cert-watcher マルチステージ Dockerfile
# 単一の静的バイナリを作り、最小ランタイムイメージで実行する。

# ---- ビルドステージ ----
FROM golang:1.23-alpine AS build
WORKDIR /src

# 依存（go.mod のみ。外部依存なしのため go.sum は無くてもよい）を先にコピーしてキャッシュ活用。
COPY go.mod ./
RUN go mod download || true

# ソースをコピーしてビルド。
COPY . .
# CGO 無効・静的リンクで OS 依存を排除（distroless/scratch で動く）。
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cert-watcher ./cmd/cert-watcher

# ---- ランタイムステージ ----
# TLS 検証用の CA 証明書を含む distroless を使用（証明書チェック対象への TLS 接続に必要）。
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/cert-watcher /usr/local/bin/cert-watcher

# 既定では check を実行。CSV はボリュームでマウントする想定。
#   docker run --rm -v "$PWD/config:/app/config" -v "$PWD/output:/app/output" \
#     cert-watcher check --input config/targets.csv --output output/report.csv --html output/report.html
ENTRYPOINT ["cert-watcher"]
CMD ["--help"]
