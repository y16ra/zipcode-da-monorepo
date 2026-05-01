# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 概要

日本郵便「郵便番号・デジタルアドレス API」を Go バックエンドでプロキシし、Next.js フロントエンドから検索するモノレポ。

## コマンド

### Docker Compose（推奨）

```bash
cp .env.example .env   # JAPANPOST_CLIENT_ID / JAPANPOST_SECRET_KEY を実値に設定してから
docker compose up --build
```

- フロントエンド: http://localhost:3000
- バックエンド: http://localhost:8080
- ヘルスチェック: `GET http://localhost:8080/healthz`

### バックエンド（ローカル）

```bash
cd backend
go run ./cmd/server          # 通常起動
air                           # ホットリロード（Air が必要）
go build ./...                # ビルド確認
go vet ./...                  # 静的解析
go test ./...                 # テスト実行
go test ./internal/handler/... # 特定パッケージのテスト
```

### フロントエンド（ローカル）

```bash
cd frontend
npm install
npm run dev    # 開発サーバー
npm run build  # プロダクションビルド
npm run lint   # ESLint
```

## アーキテクチャ

```
.
├── backend/                   # Go HTTP サーバー（標準ライブラリのみ、外部依存なし）
│   ├── cmd/server/main.go     # エントリポイント・ルーティング・CORS ミドルウェア
│   └── internal/
│       ├── config/config.go   # 環境変数の読み込みと検証
│       ├── client/japanpost.go # 日本郵便 API クライアント（トークンキャッシュ付き）
│       └── handler/
│           ├── health.go      # GET /healthz
│           └── search.go      # POST /api/v1/search/zipcode
├── frontend/                  # Next.js 16 / React 19 / TypeScript
│   └── src/app/page.tsx       # 検索フォーム（単一ページ構成）
├── docker-compose.yml         # backend + frontend の 2 サービス構成
├── go.work                    # Go ワークスペース（backend のみ）
└── .env / .env.example        # 環境変数（ルートに置き、両サービスが参照）
```

### データフロー

1. ブラウザ → `POST /api/v1/search/zipcode`（バックエンド）
2. バックエンドがトークンを取得（`POST {base}/api/v2/j/token`）、JWT をインメモリキャッシュ
3. バックエンドが `GET {base}/api/v2/searchcode/{zipcode}` を呼び出し
4. 日本郵便 API の JSON レスポンスをそのままブラウザへ返却

### 重要な実装上の注意

- **トークンキャッシュ**: `client.JapanPost` は JWT を `sync.Mutex` で保護しインメモリキャッシュ。有効期限 60 秒前に再取得する（`tokenSkew`）。
- **API パスの可変性**: トークン取得パス（`JAPANPOST_TOKEN_PATH`）と検索パス（`JAPANPOST_SEARCH_CODE_PATH`）は環境変数で変更可能。v1/v2 の切り替えに使う。
- **CORS**: `main.go` の `corsMiddleware` が `/api/v1/search/zipcode` に適用。`CORS_ALLOW_ORIGIN`（デフォルト `*`）で制御。
- **エラー処理**: 日本郵便 API の 4xx/5xx は `HTTPStatusError` 型にラップし、上流と同じ HTTP ステータスをクライアントへ返す。
- **フロントエンドの API ベース URL**: `NEXT_PUBLIC_API_BASE_URL`（デフォルト `http://localhost:8080`）でバックエンドの向き先を切り替える。

## 環境変数（必須）

| 変数 | 説明 |
|------|------|
| `JAPANPOST_CLIENT_ID` | 日本郵便 for Biz のクライアント ID |
| `JAPANPOST_SECRET_KEY` | シークレットキー |
| `JAPANPOST_API_BASE_URL` | API ベース URL（デフォルト: `https://api.da.pf.japanpost.jp`） |
| `NEXT_PUBLIC_API_BASE_URL` | フロントエンドからバックエンドへの URL |
