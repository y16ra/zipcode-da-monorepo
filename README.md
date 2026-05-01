# zipcode-da-monorepo

日本郵便「郵便番号・デジタルアドレス API」（トークン発行 + searchcode）を **Go バックエンド** から呼び出し、**Next.js** から郵便番号検索する最小構成のモノレポです。

**リリースバージョン:** `0.2.0`（[`frontend/package.json`](frontend/package.json) の `version` フィールド）

## 構成

| ディレクトリ     | 説明 |
|------------------|------|
| `backend/`       | `POST /api/v1/search/zipcode` などの HTTP API（内部で既定 **`POST /api/v2/j/token`** と **`GET /api/v2/searchcode/{code}`**。`JAPANPOST_TOKEN_PATH` / `JAPANPOST_SEARCH_CODE_PATH` で変更可） |
| `frontend/`      | **Next.js 16**（安定版。`package.json` の `next` 参照）で郵便番号・`page` / `limit` / `choikitype` / `searchtype` を指定して検索する UI |

## 前提

- **Go:** `backend/go.mod` は **Go 1.26**（`toolchain go1.26.2`）を想定。古い `go` でも `GOTOOLCHAIN=auto` ならツールチェーン取得で揃います。Docker ビルドは `golang:1.26-alpine`。
- [郵便番号・デジタルアドレス for Biz](https://guide-biz.da.pf.japanpost.jp/) で発行した **client_id** と **secret_key**
- API ベース URL（本番例: `https://api.da.pf.japanpost.jp`、スタブ等は契約・ドキュメントに従って変更）

## セットアップ

1. 環境変数ファイルを作成する。

   ```bash
   cp .env.example .env
   ```

2. `.env` の `JAPANPOST_CLIENT_ID` と `JAPANPOST_SECRET_KEY` を実値に置き換える。  
   必要に応じて `JAPANPOST_API_BASE_URL`（スタブ URL 等）も変更する。

3. Docker Compose で起動する。

   ```bash
   docker compose up --build
   ```

   - フロント: <http://localhost:3000>  
   - バックエンド: <http://localhost:8080>  
   - ヘルスチェック: `GET http://localhost:8080/healthz`

ブラウザから API を叩くため、バックエンドは `CORS_ALLOW_ORIGIN`（未設定時は `*`）で CORS を許可しています。

## ローカル開発（Docker なし）

### Backend

```bash
cd backend
export JAPANPOST_CLIENT_ID=...
export JAPANPOST_SECRET_KEY=...
export JAPANPOST_API_BASE_URL=https://api.da.pf.japanpost.jp
export JAPANPOST_X_FORWARDED_FOR=127.0.0.1
go run ./cmd/server
```

ホットリロード: [Air](https://github.com/air-verse/air) を入れたうえで `air`（設定は `backend/.air.toml`）。

### Frontend

```bash
cd frontend
cp ../.env.example ../.env   # または NEXT_PUBLIC_API_BASE_URL のみ設定
npm install
npm run dev
```

`NEXT_PUBLIC_API_BASE_URL` はブラウザがアクセスできるバックエンド URL（例: `http://localhost:8080`）に合わせる。

## 自前 API

### `GET /healthz`

疎通確認用。`200` と `{"status":"ok"}`。

### `POST /api/v1/search/zipcode`

日本郵便の **コード番号検索（searchcode）** を郵便番号で実行するプロキシ。

**Request**（`Content-Type: application/json`）

| フィールド     | 型     | 必須 | 説明 |
|----------------|--------|------|------|
| `zipcode`      | string | はい | 3〜7桁の数字（ハイフン可）。パス `/api/v1/searchcode/{zipcode}` に渡す値に正規化 |
| `page`         | number | いいえ | デフォルト `1` |
| `limit`        | number | いいえ | デフォルト `1000`、最大 `1000` |
| `choikitype`   | number | いいえ | `1`: 括弧なし町域 / `2`: 括弧あり |
| `searchtype`   | number | いいえ | `1`: 郵便+事業所個別+DA / `2`: 事業所個別を除く |
| `ec_uid`       | string | いいえ | クエリ `ec_uid`。未指定時は環境変数 `JAPANPOST_EC_UID` を使用（どちらも空なら付与しない） |

**Response**

日本郵便 API の JSON をそのまま返却（キーは仕様どおり返り、値なしは `null`）。

**エラー例**

- 郵便番号形式不正 → `400`
- 日本郵便 API が 4xx/5xx → 可能な範囲で同一ステータスを返し、`detail` に本文を含む

## 日本郵便 API（参考）

**データソースについて:** for Biz で提供される郵便番号検索結果のデータは、[郵便番号データダウンロード（UTF-8・CSV・ZIP）](https://www.post.japanpost.jp/zipcode/dl/utf-zip.html) で公開されている **住所の郵便番号（1レコード1行）** などのデータを根拠としたものです。本モノレポは日本郵便の API をプロキシするのみで、これらの CSV データはリポジトリに同梱しません。

実装は公開ドキュメントに基づく。

- トークン: `POST {base}{JAPANPOST_TOKEN_PATH}`（既定 **`/api/v2/j/token`**。リファレンスのリクエストサンプルと同じ）。**OAuth 2.0 `client_credentials`** の JSON 例: `{"grant_type":"client_credentials","client_id":"…","secret_key":"…"}`。必須ヘッダー **`x-forwarded-for`**（送信元 IP）と **`Content-Type: application/json`**、および **`User-Agent`** を付与し、成功時は **JWT** の `token` が返ります。まだ **`/api/v1/j/token`** の環境は **`JAPANPOST_TOKEN_PATH=/api/v1/j/token`** を指定。本文に `scope` が必要な場合は **`JAPANPOST_TOKEN_SCOPE`**（例: `J1`）を設定。
- 検索: `GET {base}{JAPANPOST_SEARCH_CODE_PATH}/{search_code}`（既定 **`/api/v2/searchcode`**）— `Authorization: Bearer {token}`、クエリに `page`, `limit`, `choikitype`, `searchtype`, `ec_uid` 等。トークンが v2 のホストでは検索も **v2** が必要なことがあり、v1 のみの場合は `JAPANPOST_SEARCH_CODE_PATH=/api/v1/searchcode`。
- リクエストに **`X-Forwarded-For`** と **`User-Agent`** を付与（トークン取得で拒否されないよう配慮）
- **エラー時のレスポンス（参考）:** 多くの 4xx は `application/json` で **`request_id`**（問合せ ID）、**`error_code`**、**`message`** が返る。バックエンドはこれを要約して自前 API の `detail` に載せ、可能なら上流と同じ HTTP ステータスを返す。

## トラブルシューティング

### `502` / `japanpost token: status 401` と `401-1028-0001`「スコープがありません」

トークン取得が拒否されている状態です。次を確認してください。

1. **郵便番号・デジタルアドレス for Biz** の管理画面で、利用中の **クライアント ID / シークレット** が「郵便番号・デジタルアドレス API」用として有効か（テスト用認証情報と本番のシステム登録の取り違えがないか）。
2. 組織管理者による **API 利用権限の割当**（ユーザーリスト・権限まわり）。クレデンシャルは正しくても、**スコープ（API 利用権限）が未付与**だと本メッセージになることがあります。
3. **`JAPANPOST_API_BASE_URL`** が、そのクレデンシャルが発行された環境（テスト用 URL と本番 `https://api.da.pf.japanpost.jp`）と一致しているか。
4. 契約・リファレンスで `scope` の明示が必要な場合、`.env` に `JAPANPOST_TOKEN_SCOPE=J1` を設定して再試行する。

### `404` / `[404-1029-0003]`「対応するAPIがありません」（searchcode）

トークンは取れているが **検索 URL のバージョンが合っていない**ことが多いです。API ver2.0 系では **`GET /api/v2/searchcode/{code}`** を使います。`.env` で `JAPANPOST_SEARCH_CODE_PATH=/api/v2/searchcode`（既定）になっているか確認し、古いゲートウェイだけの場合は `/api/v1/searchcode` を試してください。

### 本番 API で `403`（トークン取得時）

システム登録の **固定 IP** や **User-Agent** 要件と実リクエスト元が一致しているか、[日本郵便の Q&A](https://www.post.japanpost.jp/question/776.html) などを参照してください。

## ライセンス

リポジトリ内の [LICENSE](LICENSE) を参照。
