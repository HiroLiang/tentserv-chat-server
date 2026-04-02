# Tentserv Chat Server

<p align="center">
  <a href="../../../README.md">English</a> | <a href="../zh-TW/README.md">中文</a> | 日本語
</p>

Tentserv Chat デスクトップアプリケーション用の Go REST API + WebSocket サーバー。個人練習プロジェクト。

## 技術スタック

- **Go 1.26.1** · Gin（HTTP）· Gorilla WebSocket
- **PostgreSQL 18** · Redis 8
- **Swagger**（swaggo）· **BDD テスト**（Godog/Gherkin）

## 機能

- ユーザー認証（JWT）
- WebSocket によるリアルタイムチャット
- LLM エージェントへのリダイレクト
- E2EE 鍵配布（Signal Protocol X3DH）
- グループチャットと sender key の再キーイング

## クイックスタート

### 1. 依存サービスの起動

```shell
docker network create tentserv-net

# Redis（オプション：-v ~/data/redis:/data でデータ永続化）
docker run -d --name redis --network tentserv-net \
  -p 6379:6379 \
  docker.io/library/redis:8 \
  redis-server --appendonly yes --requirepass "1234"

# Postgres（オプション：-v ~/data/postgres:/data でデータ永続化）
docker run -d --name postgres --network tentserv-net \
  -e POSTGRES_USER=root -e POSTGRES_PASSWORD=1234 -e POSTGRES_DB=tentserv \
  -p 5432:5432 \
  docker.io/library/postgres:18
```

### 2. 設定

`config/config.yaml` を作成：

```yaml
auth_token:
  expiration: 3600
secrets:
  HMAC_SECRET: "00000000000000000000000000000000"
databases:
  postgres:
    driver: pgx
    dsn: "postgres://root:1234@localhost:5432/goat?sslmode=disable"
    config:
      max_open_conns: 30
      max_idle_conns: 15
      conn_max_lifetime: 3600
      conn_max_idle_time: 600
redis:
  addr: "localhost:6379"
  password: "1234"
  db: 0
```

`.env` を作成：

```dotenv
APP_ENV=dev
SERVER_PORT=8080
```

データベーススキーマの初期化：

```shell
psql -h localhost -U root -d goat -f config/init_postgres.sql
```

### 3. 実行

```shell
make setup   # swag と godog CLI をインストール（初回のみ）
make run     # サーバーを :8080 で起動
```

## コマンド

```shell
make build   # bin/goat-api にコンパイル
make run     # サーバーを :8080 で起動
make test    # 全テスト実行（ユニット + BDD）
make unit    # ユニットテストのみ：go test ./internal/... -v
make bdd     # BDD テストのみ：go test -v ./features
make swag    # Swagger ドキュメントを再生成
make clean   # ビルド出力を削除
```

単一テストの実行：

```shell
go test ./internal/path/to/pkg/... -v -run TestFunctionName
```

## ローカルデプロイ（Docker）

```shell
# イメージをビルド
docker build -t tentserv-chat-server:latest .

# コンテナを実行（goat-net ネットワークで DB/Redis にアクセス）
docker run -d --name tentserv-chat-server --network goat-net \
  -p 8080:8080 \
  -v ./config:/app/config:ro \
  -v .env:/app/.env:ro \
  goat-server:latest
```
