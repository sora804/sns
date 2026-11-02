// Package security はパスワードのハッシュ化などを担う。
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2Params struct {
	Memory  uint32 // KiB
	Time    uint32
	Threads uint8
	KeyLen  uint32
	SaltLen uint32
}

// DefaultParams は OWASP の推奨に沿った値。
var DefaultParams = Argon2Params{Memory: 19 * 1024, Time: 2, Threads: 1, KeyLen: 32, SaltLen: 16}

type Argon2Hasher struct{ p Argon2Params }

func NewArgon2Hasher(p Argon2Params) *Argon2Hasher { return &Argon2Hasher{p: p} }

var b64 = base64.RawStdEncoding

// Hash は "argon2id$m$t$p$salt$hash" 形式の文字列を返す。
func (h *Argon2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, h.p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, h.p.Time, h.p.Memory, h.p.Threads, h.p.KeyLen)
	return fmt.Sprintf("argon2id$%d$%d$%d$%s$%s", h.p.Memory, h.p.Time, h.p.Threads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

func (h *Argon2Hasher) Verify(hash, password string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "argon2id" {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(strings.Join(parts[1:4], " "), "%d %d %d", &m, &t, &p); err != nil {
		return false
	}
	salt, err1 := b64.DecodeString(parts[4])
	want, err2 := b64.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(want) == 0 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}
