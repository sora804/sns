package backend

// OpenAPI 仕様（../api/openapi.yml）から internal/gen を生成する。実行: go generate ./...
// ogen のバージョンは go.mod の tool ディレクティブで固定している。
//go:generate go tool ogen --target internal/gen --package gen --clean ../api/openapi.yml
