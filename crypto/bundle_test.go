package crypto

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

const vector = "7e4a8e65a8acf796ad650afcfa454f5f4a88d12d63458e566d8b8d31ac2c8a1650b08d110649b615376639b65bd87f572a48716a06518a8ec5222c8fa4087cf477b67ce46cc57cd4a64a885126251d66f2463c3b2b09ef8544d76e242281269d775afc482f6ecaeaff5bb30c82045b0b130bb8409be39a9cf68781d59b1c2f99"

func TestDecryptMatchesReference(t *testing.T) {
	data := bytes.Repeat(func() []byte {
		b := make([]byte, 64)
		for i := range b {
			b[i] = byte(i)
		}
		return b
	}(), 2)
	Decrypt(data, "sample_0001.bundle")
	if hex.EncodeToString(data) != vector {
		t.Fatal("keystream differs from reference implementation")
	}
}

func TestDecryptFileRoundTrip(t *testing.T) {
	plain := append(append([]byte{}, Magic...), bytes.Repeat([]byte("x"), 40000)...)
	enc := append([]byte{}, plain...)
	Decrypt(enc, "a.bundle")
	dir := t.TempDir()
	src := filepath.Join(dir, "a.bundle")
	dst := filepath.Join(dir, "out", "a.bundle")
	os.WriteFile(src, enc, 0644)
	if err := DecryptFile(src, dst, "a.bundle", Provider); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if !bytes.Equal(got, plain) {
		t.Fatal("round trip mismatch")
	}
	if err := DecryptFile(src, dst, "wrong.bundle", Provider); err != ErrNotBundle {
		t.Fatalf("expected ErrNotBundle, got %v", err)
	}
}
