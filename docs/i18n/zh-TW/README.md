# Tentserv Chat Server

<p align="center">
  <a href="../../../README.md">English</a> | 中文 | <a href="../ja/README.md">日本語</a>
</p>

Tentserv Chat 桌面應用的 Go REST API + WebSocket 伺服器。個人練習專案。

## 技術堆疊

- **Go 1.26.1** · Gin（HTTP）· Gorilla WebSocket
- **PostgreSQL 18** · Redis 8
- **Swagger**（swaggo）· **BDD 測試**（Godog/Gherkin）

## 功能

- 使用者驗證（JWT）
- 透過 WebSocket 即時聊天
- LLM 轉發至代理
- E2EE 金鑰分發（Signal Protocol X3DH）
- 群組聊天與 sender key 重新分鑰

## 快速開始

### 1. 啟動相依服務

```shell
docker network create tentserv-net

# Redis（可選：加入 -v ~/data/redis:/data 以持久化資料）
docker run -d --name redis --network tentserv-net \
  -p 6379:6379 \
  docker.io/library/redis:8 \
  redis-server --appendonly yes --requirepass "1234"

# Postgres（可選：加入 -v ~/data/postgres:/data 以持久化資料）
docker run -d --name postgres --network tentserv-net \
  -e POSTGRES_USER=root -e POSTGRES_PASSWORD=1234 -e POSTGRES_DB=tentserv \
  -p 5432:5432 \
  docker.io/library/postgres:18
```

### 2. 設定

建立 `config/config.yaml`：

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

建立 `.env`：

```dotenv
APP_ENV=dev
SERVER_PORT=8080
```

初始化資料庫結構：

```shell
psql -h localhost -U root -d goat -f config/init_postgres.sql
```

### 3. 執行

```shell
make setup   # 安裝 swag 與 godog CLI（僅首次需要）
make run     # 啟動伺服器於 :8080
```

## 指令

```shell
make build   # 編譯為 bin/goat-api
make run     # 啟動伺服器於 :8080
make test    # 執行所有測試（單元 + BDD）
make unit    # 僅單元測試：go test ./internal/... -v
make bdd     # 僅 BDD 測試：go test -v ./features
make swag    # 重新產生 Swagger 文件
make clean   # 移除建置輸出
```

執行單一測試：

```shell
go test ./internal/path/to/pkg/... -v -run TestFunctionName
```

## 本地部署（Docker）

```shell
# 建置映像
docker build -t tentserv-chat-server:latest .

# 執行容器（使用 goat-net 網路以存取 DB/Redis）
docker run -d --name tentserv-chat-server --network goat-net \
  -p 8080:8080 \
  -v ./config:/app/config:ro \
  -v .env:/app/.env:ro \
  goat-server:latest
```
