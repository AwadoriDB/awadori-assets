package extract

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz/lzma"
)

func be32(b *bytes.Buffer, v uint32) { binary.Write(b, binary.BigEndian, v) }
func be64(b *bytes.Buffer, v uint64) { binary.Write(b, binary.BigEndian, v) }
func le32(b *bytes.Buffer, v uint32) { binary.Write(b, binary.LittleEndian, v) }
func le64(b *bytes.Buffer, v uint64) { binary.Write(b, binary.LittleEndian, v) }

func align(b *bytes.Buffer, n int) {
	for b.Len()%n != 0 {
		b.WriteByte(0)
	}
}

func textAsset(name string, script []byte) []byte {
	var b bytes.Buffer
	le32(&b, uint32(len(name)))
	b.WriteString(name)
	align(&b, 4)
	le32(&b, uint32(len(script)))
	b.Write(script)
	align(&b, 4)
	return b.Bytes()
}

func serialized(assets map[string][]byte, names []string) []byte {
	var objs bytes.Buffer
	var meta bytes.Buffer
	le32(&meta, 0)
	var table bytes.Buffer
	for i, n := range names {
		raw := textAsset(n, assets[n])
		align(&table, 4)
		le64(&table, uint64(i+1))
		le64(&table, uint64(objs.Len()))
		le32(&table, uint32(len(raw)))
		le32(&table, 0)
		objs.Write(raw)
		for objs.Len()%8 != 0 {
			objs.WriteByte(0)
		}
	}
	var body bytes.Buffer
	body.WriteString("2021.3.16f1\x00")
	le32(&body, 13)
	body.WriteByte(0)
	le32(&body, 1)
	le32(&body, 49)
	body.WriteByte(0)
	body.Write([]byte{0xff, 0xff})
	body.Write(make([]byte, 16))
	le32(&body, uint32(len(names)))
	body.Write(table.Bytes())
	le32(&body, 0)
	le32(&body, 0)
	le32(&body, 0)
	body.WriteString("\x00")

	headerLen := 48
	dataOffset := headerLen + body.Len()
	for dataOffset%16 != 0 {
		dataOffset++
	}
	var out bytes.Buffer
	be32(&out, 0)
	be32(&out, 0)
	be32(&out, 22)
	be32(&out, 0)
	out.WriteByte(0)
	out.Write([]byte{0, 0, 0})
	be32(&out, uint32(body.Len()))
	be64(&out, uint64(dataOffset+objs.Len()))
	be64(&out, uint64(dataOffset))
	be64(&out, 0)
	out.Write(body.Bytes())
	for out.Len() < dataOffset {
		out.WriteByte(0)
	}
	out.Write(objs.Bytes())
	return out.Bytes()
}

func lz4Literal(src []byte) []byte {
	var out bytes.Buffer
	n := len(src)
	if n < 15 {
		out.WriteByte(byte(n << 4))
	} else {
		out.WriteByte(0xf0)
		rest := n - 15
		for rest >= 255 {
			out.WriteByte(255)
			rest -= 255
		}
		out.WriteByte(byte(rest))
	}
	out.Write(src)
	return out.Bytes()
}

func lzmaBlock(t *testing.T, src []byte) []byte {
	var buf bytes.Buffer
	cfg := lzma.WriterConfig{SizeInHeader: true, Size: int64(len(src))}
	w, err := cfg.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	w.Write(src)
	w.Close()
	raw := buf.Bytes()
	return append(append([]byte{}, raw[:5]...), raw[13:]...)
}

func archive(t *testing.T, mode uint32, payload []byte) []byte {
	pack := func(b []byte) []byte {
		switch mode {
		case 1:
			return lzmaBlock(t, b)
		case 2:
			return lz4Literal(b)
		}
		return b
	}
	var info bytes.Buffer
	info.Write(make([]byte, 16))
	be32(&info, 1)
	compressed := pack(payload)
	be32(&info, uint32(len(payload)))
	be32(&info, uint32(len(compressed)))
	binary.Write(&info, binary.BigEndian, uint16(mode))
	be32(&info, 1)
	be64(&info, 0)
	be64(&info, uint64(len(payload)))
	be32(&info, 4)
	info.WriteString("CAB-test\x00")
	packedInfo := pack(info.Bytes())

	var out bytes.Buffer
	out.WriteString("UnityFS\x00")
	be32(&out, 7)
	out.WriteString("5.x.x\x002021.3.16f1\x00")
	be64(&out, 0)
	be32(&out, uint32(len(packedInfo)))
	be32(&out, uint32(info.Len()))
	be32(&out, mode)
	align(&out, 16)
	out.Write(packedInfo)
	out.Write(compressed)
	return out.Bytes()
}

func TestExtractModes(t *testing.T) {
	names := []string{"0001_00", "note.txt", "big"}
	assets := map[string][]byte{
		"0001_00":  []byte(`{"notes":[1,2,3]}`),
		"note.txt": []byte("hello"),
		"big":      bytes.Repeat([]byte("abcdefgh"), 5000),
	}
	payload := serialized(assets, names)
	for _, mode := range []uint32{0, 1, 2} {
		dir := t.TempDir()
		src := filepath.Join(dir, "bundle.bundle")
		if err := os.WriteFile(src, archive(t, mode, payload), 0644); err != nil {
			t.Fatal(err)
		}
		if mode == 2 {
			os.WriteFile("/tmp/awano_test_lz4.bundle", archive(t, mode, payload), 0644)
		}
		res, err := File(src, filepath.Join(dir, "out"))
		if err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
		if len(res.Written) != 3 {
			t.Fatalf("mode %d: wrote %d files", mode, len(res.Written))
		}
		for n, want := range map[string][]byte{"0001_00.bytes": assets["0001_00"], "note.txt": assets["note.txt"], "big.bytes": assets["big"]} {
			got, err := os.ReadFile(filepath.Join(dir, "out", n))
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("mode %d: %s mismatch (%v)", mode, n, err)
			}
		}
	}
}

func TestRejectsPlainFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "x.bin")
	os.WriteFile(src, []byte("not a unity file at all, just text"), 0644)
	if _, err := File(src, dir); err == nil {
		t.Fatal("expected error")
	}
}
