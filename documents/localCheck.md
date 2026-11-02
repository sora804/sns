# ローカルでの動作確認手順

バックエンド（Go）・フロントエンド（React）・PostgreSQL をローカルで動かして、ブラウザで確認する手順。

## 0. 前提

| ツール | 確認コマンド | 備考 |
|--------|--------------|------|
| Go | `go version` | `backend/go.mod` の版以上 |
| Node.js / npm | `node --version` | 22 以上を想定 |
| PostgreSQL | 下の「1」のどちらか | Docker か、ローカルにインストールしたもの |

リポジトリのルート（`pythonProject`）から始める。

## 1. データベースを用意する（どちらか一方）

### 方法A: Docker（簡単）

Docker Desktop を起動してから実行する。

```bash
docker compose up -d db
```

接続先は次のとおり。

```
postgres://sns:sns@localhost:5432/sns?sslmode=disable
```

### 方法B: Docker を使わない（ローカルの PostgreSQL を使う）

すでに PostgreSQL が入っている場合でも、**専用のクラスタを別に作る**と、既存の DB に影響しない。
以下は Git Bash での例（PostgreSQL 18 を `C:\Program Files\PostgreSQL\18` に入れた場合）。

```bash
export PATH="/c/Program Files/PostgreSQL/18/bin:$PATH"

# 初回だけ: 専用のクラスタを作る（ポート 5433、パスワードなしでローカルからのみ接続）
mkdir -p .localdb && cd .localdb
initdb -D pgdata -U sns --auth=trust -E UTF8 --locale=C
pg_ctl -D pgdata -o "-p 5433 -c listen_addresses=127.0.0.1" -l pg.log -w start
createdb -h 127.0.0.1 -p 5433 -U sns sns        # 開発用
createdb -h 127.0.0.1 -p 5433 -U sns sns_test   # テスト用
cd ..
```

2回目以降は、起動だけ行う。

```bash
export PATH="/c/Program Files/PostgreSQL/18/bin:$PATH"
pg_ctl -D .localdb/pgdata -o "-p 5433 -c listen_addresses=127.0.0.1" -l .localdb/pg.log -w start
```

接続先は次のとおり。

```
postgres://sns@127.0.0.1:5433/sns?sslmode=disable
```

`.localdb/` は `.gitignore` 済み。**この方法は認証なしなので、ローカル確認専用**にする。

## 2. バックエンドを起動する

別のターミナルで実行する。`DATABASE_URL` は「1」で使った接続先に合わせる。

```bash
cd backend
DATABASE_URL="postgres://sns@127.0.0.1:5433/sns?sslmode=disable" COOKIE_SECURE=false go run ./cmd/server
```

PowerShell の場合は次のとおり。

```powershell
cd backend
$env:DATABASE_URL = "postgres://sns@127.0.0.1:5433/sns?sslmode=disable"
$env:COOKIE_SECURE = "false"
go run ./cmd/server
```

- 初回は起動時にテーブルが自動で作られる（マイグレーション）。
- `COOKIE_SECURE=false` は、HTTP（`http://localhost`）でログインできるようにするための設定。本番では付けない。
- 動作確認: `curl http://127.0.0.1:8080/healthz` が 200 を返す。

## 3. フロントエンドを起動する

さらに別のターミナルで実行する。

```bash
cd frontend
npm install     # 初回のみ
npm run dev
```

ブラウザで **http://localhost:5173** を開く。`/api` へのリクエストは、自動で backend（`:8080`）へ中継される。

## 4. 画面で確認すること

1. 「新規登録」からアカウントを作る（登録後は自動でログインされる）。
   - ユーザーID: 3〜15文字の英数字とアンダースコア
   - パスワード: 8文字以上
2. ホームで投稿する（140文字まで。残り文字数が表示される）。
3. 投稿の「♡」でいいね、「削除」で削除できる。
4. 自分の `@ユーザーID` からプロフィールを開く。
5. **フォローの確認**: シークレットウィンドウなど別のブラウザで、別のアカウントを登録する。
   - そのアカウントで、最初のユーザーのプロフィール（`http://localhost:5173/users/ユーザーID`）を開き、「フォローする」を押す。
   - ホームに、フォローしたユーザーの投稿が表示される。
6. 「ログアウト」後は、ホームを開くとログイン画面に移る。

## 5. テストを実行する

```bash
# バックエンド（DBなしのテスト）
cd backend && go test ./...

# バックエンド（DBを使う結合テスト。sns_test を空にするので、開発用DBは指定しない）
TEST_DATABASE_URL="postgres://sns@127.0.0.1:5433/sns_test?sslmode=disable" go test ./...

# フロントエンド
cd frontend && npm test && npm run lint && npm run build
```

## 6. API の仕様を変えたとき

`api/openapi.yml` を編集したら、コードと型を再生成する。

```bash
cd backend  && go generate ./...      # internal/gen（ogen）
cd frontend && npm run gen:api        # src/api/schema.d.ts
```

## 7. 停止とデータの削除

- backend / frontend: 各ターミナルで `Ctrl + C`。
- PostgreSQL（方法A）: `docker compose down`（データも消すなら `docker compose down -v`）。
- PostgreSQL（方法B）:
  ```bash
  export PATH="/c/Program Files/PostgreSQL/18/bin:$PATH"
  pg_ctl -D .localdb/pgdata -m fast stop
  rm -rf .localdb        # データを完全に消す場合
  ```

## 8. うまくいかないとき

| 症状 | 原因と対処 |
|------|------------|
| backend が `DATABASE_URL が設定されていません` で終了する | 環境変数を設定していない。「2」のコマンドをそのまま使う |
| backend が DB に接続できない | DB が起動していない、またはポートが違う。「1」を確認する |
| ログインしてもすぐログアウトされる | `COOKIE_SECURE=false` を付けずに HTTP で動かしている |
| 画面は出るが API が 502 / 接続エラー | backend が起動していない（`:8080`） |
| ポート 5433 が使用中 | 方法Bのポートを変え、`DATABASE_URL` も合わせる |
| 新規登録で「すでに使われています」 | ユーザーIDまたはメールが登録済み。別の値にする |
| `npm run gen:api` が `createKeywordTypeNode` で失敗する | TypeScript が 7.x になっている。5.x に固定する（`npm install -D typescript@~5.9`） |
