# backend

Go製のAPIサーバー。設計は [documents/アーキテクチャ.md](../documents/アーキテクチャ.md)、API仕様は [api/openapi.yml](../api/openapi.yml) を参照。

## 実装済み（フェーズ1の一部）

アカウント（登録・ログイン・ログアウト）、投稿（作成・取得・削除）、ホームタイムライン、フォロー、いいね、プロフィール。

## コード生成（ogen）

`api/openapi.yml` から `internal/gen` を生成している。**仕様を変えたら再生成する**（生成物はコミットする）。

```bash
cd backend
go generate ./...
```

`internal/gen` は手で編集しない。API の追加・変更は、仕様 → 再生成 → `internal/handler` の実装、の順で行う。

## 起動

```bash
# PostgreSQL を起動
docker compose up -d db

# サーバーを起動（http で試すので COOKIE_SECURE=false）
cd backend
DATABASE_URL="postgres://sns:sns@localhost:5432/sns?sslmode=disable" COOKIE_SECURE=false go run ./cmd/server
```

起動時に `migrations/` の未適用SQLが自動で適用される。

| 環境変数 | 既定値 | 説明 |
|----------|--------|------|
| `DATABASE_URL` | （必須） | PostgreSQL の接続文字列 |
| `PORT` | `8080` | 待ち受けポート |
| `COOKIE_SECURE` | `true` | セッションCookieに Secure を付ける。HTTPでの動作確認時だけ `false` |

## テスト

```bash
go test ./...
```

DBを使う結合テストは `TEST_DATABASE_URL` を設定したときだけ動く（未設定ならスキップ）。
**テーブルを空にするので、開発用DBとは別のDBを指定すること。**

```bash
TEST_DATABASE_URL="postgres://sns:sns@localhost:5432/sns_test?sslmode=disable" go test ./...
```

## 未実装・今後

- レート制限、返信、画像添付、通知、検索、通報・管理画面
