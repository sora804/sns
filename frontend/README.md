# frontend

React + TypeScript + Vite の SPA。設計は [documents/アーキテクチャ.md](../documents/アーキテクチャ.md) を参照。

## 実装済み

登録・ログイン・ログアウト、ホームタイムライン（もっと見る）、投稿・削除、いいね、プロフィール、フォロー、投稿詳細。

## 起動

```bash
cd frontend
npm install
npm run dev      # http://localhost:5173 （/api は backend の :8080 へ中継）
```

backend も起動しておくこと（[backend/README.md](../backend/README.md)）。

## OpenAPI からの型生成

`src/api/schema.d.ts` は **`api/openapi.yml` から自動生成**する（手で編集しない。生成物はコミットする）。
API の仕様を変えたら再生成する。

```bash
npm run gen:api
```

API 呼び出しは `src/api/client.ts`（openapi-fetch）経由で行い、パスやレスポンスの型は生成された型で検査される。

## コマンド

| コマンド | 内容 |
|----------|------|
| `npm run dev` | 開発サーバー |
| `npm run build` | 型チェック + 本番ビルド（`dist/`） |
| `npm run lint` | 型チェック |
| `npm test` | テスト（Vitest） |

## 構成

```
src/
├── api/         client.ts（API クライアント）、hooks.ts（TanStack Query）、schema.d.ts（自動生成）
├── components/  Layout, PostCard, PostForm, Timeline
├── pages/       AuthPages（ログイン・登録）, Pages（ホーム・プロフィール・投稿詳細）
└── App.tsx      ルーティング
```

## 注意

- TypeScript は 5.x に固定している。openapi-typescript が TypeScript 7 の API に未対応のため。
