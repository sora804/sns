// Package migrations はDBマイグレーション（SQL）をバイナリに埋め込む。
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
