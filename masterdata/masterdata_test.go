package masterdata

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	testIV  = "6465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f80818283"
	vecCT   = "e13b032e112a32b579080f08b1f7ed4c2e5d3a07f97f21ee232d178a209af6b5887f66e8092402aa49f2c1551b27fe53266e490db138489ce814d58d145a8b4f994fed15c5b2fdaeeff317f157e1e0978c3f5fd5df3d34f8c08262b03750894f"
	vecPT   = "35e1121040875b488f0a9b113a53097913f60ca8b52137e3d44a247d3b308fc6232d6f5d16abadc4591df8dfc6c606cc4d1d1620338c7b8a4626d0f19877e523ae029c7c9d2b2625007938092b4bd98290b79c022d8bd6c73e3e27d4f95c9186"
)

func mustKey(t *testing.T) *Key {
	k, err := NewKey(testKey, testIV)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestDecryptMatchesReferenceVector(t *testing.T) {
	k := mustKey(t)
	ct, _ := hex.DecodeString(vecCT)
	got, err := k.Cipher.DecryptCBC(ct, k.IV)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(got) != vecPT {
		t.Fatal("Rijndael-256 CBC output differs from the reference implementation")
	}
}

func TestEncryptMatchesReferenceVector(t *testing.T) {
	k := mustKey(t)
	pt, _ := hex.DecodeString(vecPT)
	got, err := k.Cipher.EncryptCBC(pt, k.IV)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(got) != vecCT {
		t.Fatal("Rijndael-256 CBC encryption does not invert the reference vector")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	k := mustKey(t)
	for _, size := range []int{0, 1, 31, 32, 33, 5000} {
		doc := []byte(`{"_allData":[` + strings.Repeat(`{"_id":1,"_name":"x"},`, size) + `{"_id":0}]}`)
		raw, err := Encode(doc, k, nil)
		if err != nil {
			t.Fatal(err)
		}
		if (len(raw)-PrefixSize)%BlockSize != 0 {
			t.Fatalf("ciphertext length %d is not block aligned", len(raw)-PrefixSize)
		}
		back, err := Decode(raw, k)
		if err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		if !bytes.Equal(back, doc) {
			t.Fatalf("size %d: round trip mismatch", size)
		}
	}
}

func TestDecodeRejectsWrongKey(t *testing.T) {
	k := mustKey(t)
	raw, _ := Encode([]byte(`{"_allData":[]}`), k, nil)
	other, _ := NewKey(strings.Repeat("11", 32), testIV)
	if _, err := Decode(raw, other); err == nil {
		t.Fatal("wrong key must not decode")
	}
	if _, err := Decode(raw[:PrefixSize], k); err != ErrShortFile {
		t.Fatalf("expected ErrShortFile, got %v", err)
	}
}

func TestEncodeRejectsInvalidJSON(t *testing.T) {
	if _, err := Encode([]byte(`{"_allData":`), mustKey(t), nil); err != ErrBadJSON {
		t.Fatalf("expected ErrBadJSON, got %v", err)
	}
}

func TestPrefixIsPreserved(t *testing.T) {
	k := mustKey(t)
	prefix := bytes.Repeat([]byte{0xab}, PrefixSize)
	raw, _ := Encode([]byte(`{"_allData":[]}`), k, prefix)
	if !bytes.Equal(raw[:PrefixSize], prefix) {
		t.Fatal("prefix was not written")
	}
}

func TestDirRoundTripAndManifest(t *testing.T) {
	k := mustKey(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "json")
	os.MkdirAll(src, 0755)
	os.WriteFile(filepath.Join(src, "Card.json"), []byte("{\n \"_allData\": [{\"_id\": 1}]\n}"), 0644)
	os.WriteFile(filepath.Join(src, "Music.json"), []byte(`{"_allData":[{"_id":2}]}`), 0644)

	bin := filepath.Join(dir, "bin")
	res, err := EncodeDir(src, bin, k, "", 4)
	if err != nil || res.Done != 2 || len(res.Failed) != 0 {
		t.Fatalf("encode: %+v %v", res, err)
	}
	m, err := BuildManifest(bin, "9.9.9")
	if err != nil || len(m.Files) != 2 || m.Files[0].Name != "Card.bin" {
		t.Fatalf("manifest: %+v %v", m, err)
	}
	if err := WriteManifest(bin, m); err != nil {
		t.Fatal(err)
	}

	back := filepath.Join(dir, "back")
	res, err = DecodeDir(bin, back, k, false, 4)
	if err != nil || res.Done != 2 {
		t.Fatalf("decode: %+v %v", res, err)
	}
	got, _ := os.ReadFile(filepath.Join(back, "Card.json"))
	if string(got) != `{"_allData":[{"_id":1}]}` {
		t.Fatalf("unexpected decoded content %q", got)
	}
}

func TestDownloadVerifiesHashes(t *testing.T) {
	k := mustKey(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "json")
	os.MkdirAll(src, 0755)
	os.WriteFile(filepath.Join(src, "Card.json"), []byte(`{"_allData":[]}`), 0644)
	bin := filepath.Join(dir, "bin")
	EncodeDir(src, bin, k, "", 1)
	m, _ := BuildManifest(bin, "1.2.3")
	WriteManifest(bin, m)

	corrupt := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if !strings.HasPrefix(r.URL.Path, "/master/1.2.3/") {
			http.NotFound(w, r)
			return
		}
		data, err := os.ReadFile(filepath.Join(bin, name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if corrupt && name == "Card.bin" {
			data[len(data)-1] ^= 1
		}
		w.Write(data)
	}))
	defer srv.Close()

	out := filepath.Join(dir, "dl")
	res, err := Download(srv.URL, "1.2.3", out, 2)
	if err != nil || res.Downloaded != 1 || len(res.Failed) != 0 {
		t.Fatalf("download: %+v %v", res, err)
	}
	res, err = Download(srv.URL, "1.2.3", out, 2)
	if err != nil || res.Kept != 1 || res.Downloaded != 0 {
		t.Fatalf("second download must keep the file: %+v %v", res, err)
	}

	corrupt = true
	res, err = Download(srv.URL, "1.2.3", filepath.Join(dir, "dl2"), 2)
	if err != nil || len(res.Failed) != 1 {
		t.Fatalf("corrupt file must fail: %+v %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "dl2", ManifestName)); err == nil {
		t.Fatal("manifest must not be written after a failed download")
	}
}
