package crypto

import (
	"bytes"
	"encoding/hex"
	"io"
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

func TestEncryptIsInverseOfDecrypt(t *testing.T) {
	plain := append(append([]byte{}, Magic...), bytes.Repeat([]byte("y"), 50000)...)
	enc := append([]byte{}, plain...)
	Encrypt(enc, "b.bundle")
	if bytes.Equal(enc[:Window], plain[:Window]) || !bytes.Equal(enc[Window:], plain[Window:]) {
		t.Fatal("only the first window must change")
	}
	Decrypt(enc, "b.bundle")
	if !bytes.Equal(enc, plain) {
		t.Fatal("decrypt(encrypt(x)) != x")
	}
}

func TestEncryptFileAndReader(t *testing.T) {
	plain := append(append([]byte{}, Magic...), bytes.Repeat([]byte("z"), 40000)...)
	dir := t.TempDir()
	dec := filepath.Join(dir, "dec.bundle")
	os.WriteFile(dec, plain, 0644)

	enc := filepath.Join(dir, "out", "n.bundle")
	if err := EncryptFile(dec, enc, "n.bundle", Provider); err != nil {
		t.Fatal(err)
	}
	want, _ := os.ReadFile(enc)
	back := filepath.Join(dir, "back.bundle")
	if err := DecryptFile(enc, back, "n.bundle", Provider); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(back); !bytes.Equal(got, plain) {
		t.Fatal("EncryptFile -> DecryptFile mismatch")
	}
	if err := EncryptFile(enc, filepath.Join(dir, "x"), "n.bundle", Provider); err != ErrAlreadyEncrypted {
		t.Fatalf("expected ErrAlreadyEncrypted, got %v", err)
	}

	pass := filepath.Join(dir, "pass.bundle")
	if err := EncryptFile(dec, pass, "n.bundle", "Other"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(pass); !bytes.Equal(got, plain) {
		t.Fatal("non-crypt provider must be copied unchanged")
	}

	r, err := OpenEncrypted(dec, "n.bundle", Provider)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	all := make([]byte, r.Size())
	if _, err := io.NewSectionReader(r, 0, r.Size()).Read(all); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	if !bytes.Equal(all[:Window+100], want[:Window+100]) {
		t.Fatal("reader output differs from EncryptFile output")
	}
	mid := make([]byte, 200)
	if _, err := r.ReadAt(mid, Window-100); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mid, want[Window-100:Window+100]) {
		t.Fatal("ReadAt across window boundary mismatch")
	}
}
