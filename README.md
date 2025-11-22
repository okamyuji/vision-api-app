# Vision API Server (Go版 - Clean Architecture)

Claude Vision APIを使用した画像認識Web APIサーバー（Go実装、Clean Architecture設計）

## 概要

Claude Vision API（Haiku 4.5）を使用して画像からテキストを抽出するRESTful API。Clean Architectureに基づいた設計により、高いテスタビリティと保守性を実現。

## 特徴

- **Claude Vision API**: Claude Haiku 4.5による高精度な画像認識
- **用途別プロンプト最適化**:
    - レシート読み取り専用プロンプト（JSON形式で構造化データ抽出）
    - 家計簿仕訳け専用プロンプト（カテゴリ自動判定）
    - 汎用テキスト抽出プロンプト
- **RESTful API**: 標準の`net/http`のみを使用したシンプルな設計
- **Clean Architecture設計**: レイヤー分離、依存性注入(DI)による疎結合
- **Prompt Caching**: Claude APIのプロンプトキャッシュ機能を活用
- **Redis Caching**: アプリケーション層での24時間キャッシュ
- **MySQL Database**: レシート・家計簿データの永続化
- **Docker対応**: コンテナ化による環境依存の解決
- **高いテストカバレッジ**: 80%以上のユニットテストカバレッジ
- **構造化ログ**: log/slogによる構造化ログ出力

## アーキテクチャ

```text
┌─────────────────────────────────────────────┐
│ Presentation Layer                          │
│  - HTTP Handlers (net/http)                 │
│  - Middleware (CORS, Logging, Recovery)     │
│  - DI Container                             │
├─────────────────────────────────────────────┤
│ UseCase Layer                               │
│  - AI Correction UseCase                    │
├─────────────────────────────────────────────┤
│ Domain Layer                                │
│  - Entities (Receipt, ExpenseEntry, etc.)   │
│  - Repository Interfaces                    │
│  - Domain Services (Validator)              │
├─────────────────────────────────────────────┤
│ Infrastructure Layer                        │
│  - Claude Repository (Vision API)           │
│  - BUN Repository (MySQL)                   │
│  - Redis Repository (Cache)                 │
└─────────────────────────────────────────────┘
```

## クイックスタート（Docker推奨）

### 前提条件

- Docker & Docker Compose
- ANTHROPIC_API_KEY環境変数（必須）

### ビルドと実行

```bash
# .envファイルを作成
cat > .env << EOF
ANTHROPIC_API_KEY=your-api-key-here
MYSQL_ROOT_PASSWORD=rootpass
EOF

# Docker Composeで起動（MySQL, Redis, APIサーバー）
docker compose up -d

# ログ確認
docker compose logs -f app

# 停止
docker compose down

# データも削除する場合
docker compose down -v
```

### APIの使用

#### 1. ヘルスチェック

```bash
curl http://localhost:8080/health
```

#### 2. 汎用画像認識（Vision API）

```bash
curl -X POST http://localhost:8080/api/v1/vision/analyze \
  -F "image=@document.png"

# レスポンス例
{
  "success": true,
  "text": "抽出されたテキスト...",
  "tokens": {
    "input_tokens": 1250,
    "output_tokens": 320,
    "total_tokens": 1570
  }
}
```

#### 3. レシート認識（構造化データ抽出）

```bash
curl -X POST http://localhost:8080/api/v1/vision/receipt \
  -F "image=@receipt.png"

# レスポンス例（JSON形式で構造化データを返す）
{
  "success": true,
  "text": "{\"store_name\":\"スーパーマーケット\",\"purchase_date\":\"2025-11-22 14:30\",\"total_amount\":1500,\"tax_amount\":150,\"items\":[{\"name\":\"野菜\",\"quantity\":1,\"price\":500}]}",
  "tokens": {
    "input_tokens": 1350,
    "output_tokens": 280,
    "total_tokens": 1630
  }
}
```

#### 4. レシートカテゴリ判定（家計簿仕訳け）

```bash
curl -X POST http://localhost:8080/api/v1/vision/categorize \
  -H "Content-Type: application/json" \
  -d '{
    "receipt_info": "{\"store_name\":\"スーパーマーケット\",\"items\":[{\"name\":\"野菜\"}]}"
  }'

# レスポンス例
{
  "success": true,
  "text": "{\"category\":\"食費\",\"confidence\":0.95,\"reason\":\"スーパーマーケットでの食品購入\"}",
  "tokens": {
    "input_tokens": 120,
    "output_tokens": 80,
    "total_tokens": 200
  }
}
```

### サービス構成

Docker Composeで以下のサービスが起動します：

- **app**: Vision API Server (ポート: 8080)
- **mysql**: MySQL 8.0 (ポート: 3306)
- **redis**: Redis 7 (ポート: 6379)

## ローカル開発

### 必要な環境

#### Go環境

```bash
# Go 1.23以上
go version

# 依存関係のインストール
go mod download
```

#### 外部サービス（ローカル開発用）

```bash
# MySQL
docker run -d --name mysql \
  -e MYSQL_ROOT_PASSWORD=rootpass \
  -e MYSQL_DATABASE=household \
  -p 3306:3306 \
  mysql:8.0

# Redis
docker run -d --name redis \
  -p 6379:6379 \
  redis:7-alpine
```

### ビルド・実行

```bash
# 環境変数の設定
export ANTHROPIC_API_KEY=your-api-key-here
export MYSQL_ROOT_PASSWORD=rootpass
export PORT=8080

# ビルド
go build -o vision-api cmd/app/main.go

# 実行
./vision-api
```

## 設定

`config.yaml` で設定をカスタマイズ可能:

```yaml
anthropic:
  api_key: ${ANTHROPIC_API_KEY}
  model: claude-haiku-4-5-20251001
  max_tokens: 4096

redis:
  host: redis
  port: 6379
  password: ""
  db: 0

mysql:
  host: mysql
  port: 3306
  user: root
  password: ${MYSQL_ROOT_PASSWORD}
  database: household
```

### 環境変数

- `ANTHROPIC_API_KEY`: Claude APIキー（必須）
- `MYSQL_ROOT_PASSWORD`: MySQLルートパスワード（デフォルト: rootpass）
- `PORT`: サーバーポート（デフォルト: 8080）

## 開発

### プロジェクト構造

```text
.
├── cmd/
│   └── app/
│       ├── main.go              # エントリーポイント
│       └── main_test.go         # Seamパターンによるテスト
├── internal/
│   ├── domain/                  # Domain Layer
│   │   ├── entity/              # エンティティ（Receipt, ExpenseEntry, etc.）
│   │   ├── repository/          # リポジトリインターフェース
│   │   └── service/             # ドメインサービス（Validator）
│   ├── usecase/                 # UseCase Layer
│   │   └── ai_correction_usecase.go
│   ├── infrastructure/          # Infrastructure Layer
│   │   ├── ai/                  # Claude Vision API実装
│   │   ├── database/            # BUN ORM実装（MySQL）
│   │   └── cache/               # Redis実装
│   ├── presentation/            # Presentation Layer
│   │   ├── http/                # HTTP API
│   │   │   ├── handler/         # HTTPハンドラー
│   │   │   ├── middleware/      # ミドルウェア（log/slog使用）
│   │   │   └── router/          # ルーター
│   │   └── di/                  # DIコンテナ
│   └── config/                  # 設定管理
├── scripts/
│   └── init.sql                 # MySQL初期化スクリプト
├── testdata/                    # テストデータ
├── Dockerfile                   # Docker設定
├── compose.yml                  # Docker Compose設定
├── Makefile                     # ビルドタスク
├── config.yaml                  # デフォルト設定
├── go.mod
├── go.sum
└── README.md
```

### テスト

```bash
# すべてのテストを実行
go test ./...

# カバレッジレポート
go test -cover ./...

# 詳細なカバレッジ
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# TestContainers使用（Docker必須）
# MySQL, Redisコンテナが自動起動・再利用されます
go test ./internal/infrastructure/database/...
go test ./internal/infrastructure/cache/...
```

### Lint

```bash
# Lint実行
golangci-lint run ./...

# または Makefileを使用
make lint
```

### Makeコマンド

```bash
make help              # ヘルプ表示
make docker-build      # Dockerイメージビルド（--no-cache）
make docker-run        # Docker実行
make test              # ローカルテスト
make test-coverage     # カバレッジレポート
make lint              # Lint実行
make clean             # クリーンアップ
```

## テストカバレッジ

- **cmd/app**: 66.7%
- **config**: 76.5%
- **domain/entity**: 100.0%
- **domain/service**: 90.9%
- **usecase**: 100.0%
- **infrastructure/ai**: 66.1%
- **infrastructure/cache**: 71.4%
- **infrastructure/database**: 79.3%
- **presentation/di**: 100.0%
- **presentation/http/handler**: 94.6%
- **presentation/http/middleware**: 100.0%
- **presentation/http/router**: 100.0%

**全体**: **80%以上達成**

## 技術スタック

- **言語**: Go 1.23+
- **アーキテクチャ**: Clean Architecture
- **AI**: Anthropic Claude API (Haiku 4.5)
- **データベース**: MySQL 8.0 (BUN ORM)
- **キャッシュ**: Redis 7
- **ログ**: log/slog (構造化ログ)
- **テスト**: TestContainers (MySQL, Redis)
- **設定**: YAML (gopkg.in/yaml.v3)
- **コンテナ**: Docker & Docker Compose

## トラブルシューティング

### Docker Composeが起動しない

```bash
# ログ確認
docker compose logs

# コンテナ再起動
docker compose down
docker compose up -d

# ボリューム削除して再起動
docker compose down -v
docker compose up -d
```

### ANTHROPIC_API_KEYエラー

```bash
# .envファイルを確認
cat .env

# 環境変数を設定
export ANTHROPIC_API_KEY=your-api-key-here

# Docker Composeで再起動
docker compose down
docker compose up -d
```

### MySQLに接続できない

```bash
# MySQLコンテナの状態確認
docker compose ps mysql

# MySQLログ確認
docker compose logs mysql

# 手動接続テスト
docker exec -it vision-mysql mysql -u root -p
```

### Redisに接続できない

```bash
# Redisコンテナの状態確認
 docker compose ps redis

# Redisログ確認
 docker compose logs redis

# 手動接続テスト
docker exec -it vision-redis redis-cli ping
```

### TestContainersが動作しない

TestContainersはDockerが必要です：

```bash
# Dockerが起動しているか確認
docker ps

# TestContainers用の環境変数（オプション）
export TESTCONTAINERS_RYUK_DISABLED=true
```

## コントリビューション

プルリクエストを歓迎します！

1. このリポジトリをフォーク
2. フィーチャーブランチを作成 (`git checkout -b feature/amazing-feature`)
3. 変更をコミット (`git commit -m 'feat: add amazing feature'`)
4. ブランチにプッシュ (`git push origin feature/amazing-feature`)
5. プルリクエストを作成

### コミットメッセージ規約

```text
<type>: <subject>

<body>

<footer>
```

**Type:**

- `feat`: 新機能
- `fix`: バグ修正
- `docs`: ドキュメント
- `style`: フォーマット
- `refactor`: リファクタリング
- `test`: テスト
- `chore`: その他

## ライセンス

MIT License

## サポート

- **Issues**: [GitHub Issues](https://github.com/yujiokamoto/tesseract-ocr-app/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yujiokamoto/tesseract-ocr-app/discussions)

## 参考資料

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Effective Go](https://golang.org/doc/effective_go)
- [Anthropic Claude API](https://docs.anthropic.com/)
- [BUN ORM](https://bun.uptrace.dev/)
- [Redis](https://redis.io/docs/)
- [TestContainers Go](https://golang.testcontainers.org/)
- [log/slog](https://pkg.go.dev/log/slog)
- [Docker](https://www.docker.com/)

---

**最終更新**: 2025-11-22  
**バージョン**: 3.0.0  
**作成者**: Yuji Okamoto
