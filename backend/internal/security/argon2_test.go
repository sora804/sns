package security

import "testing"

func TestHashAndVerify(t *testing.T) {
	h := NewArgon2Hasher(Argon2Params{Memory: 8, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8})
	hash, err := h.Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !h.Verify(hash, "correct horse") {
		t.Error("正しいパスワードが拒否された")
	}
	if h.Verify(hash, "wrong") {
		t.Error("誤ったパスワードが許可された")
	}
	if h.Verify("garbage", "x") {
		t.Error("不正な形式が許可された")
	}
	other, _ := h.Hash("correct horse")
	if hash == other {
		t.Error("ソルトが毎回同じ")
	}
}
