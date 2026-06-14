package preview

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func loadKeyframeFixture(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "web", "test-fixtures", "preview", "keyframe.h264")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read keyframe fixture: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("keyframe fixture is empty")
	}
	return data
}

func TestPackChunkKeyframe(t *testing.T) {
	annexB := loadKeyframeFixture(t)
	avcc := AnnexBToAVCC(annexB)
	if len(avcc) == 0 {
		t.Fatal("AnnexBToAVCC produced empty output")
	}

	const pts = uint64(1_234_567_890)
	chunk := PackChunk(avcc, pts, true)

	if len(chunk) != chunkHeaderSize+len(avcc) {
		t.Fatalf("chunk length = %d, want %d", len(chunk), chunkHeaderSize+len(avcc))
	}
	if chunk[0] != flagKeyframe {
		t.Fatalf("flags = %#x, want %#x", chunk[0], flagKeyframe)
	}
	if got := binary.BigEndian.Uint64(chunk[1:9]); got != pts {
		t.Fatalf("pts_us = %d, want %d", got, pts)
	}
	if got := binary.BigEndian.Uint32(chunk[9:13]); got != uint32(len(avcc)) {
		t.Fatalf("nal_len = %d, want %d", got, len(avcc))
	}
	if got := chunk[chunkHeaderSize:]; string(got) != string(avcc) {
		t.Fatal("h264 access unit does not match avcc payload")
	}
}

func TestPackChunkNonKeyframe(t *testing.T) {
	avcc := []byte{0, 0, 0, 1, 0x21}
	chunk := PackChunk(avcc, 42, false)

	if chunk[0] != 0 {
		t.Fatalf("flags = %#x, want 0", chunk[0])
	}
	if got := binary.BigEndian.Uint64(chunk[1:9]); got != 42 {
		t.Fatalf("pts_us = %d, want 42", got)
	}
	if got := binary.BigEndian.Uint32(chunk[9:13]); got != uint32(len(avcc)) {
		t.Fatalf("nal_len = %d, want %d", got, len(avcc))
	}
}

func TestAvccIsKeyframe(t *testing.T) {
	annexB := loadKeyframeFixture(t)
	avcc := AnnexBToAVCC(annexB)
	if !AvccIsKeyframe(avcc) {
		t.Fatal("fixture keyframe avcc should be detected as keyframe")
	}
	delta := []byte{0, 0, 0, 1, 0x21}
	if AvccIsKeyframe(delta) {
		t.Fatal("P-slice only avcc should not be keyframe")
	}
}

func TestAnnexBToAVCCNALLengths(t *testing.T) {
	annexB := loadKeyframeFixture(t)
	avcc := AnnexBToAVCC(annexB)

	off := 0
	nalCount := 0
	for off+4 <= len(avcc) {
		nalLen := binary.BigEndian.Uint32(avcc[off : off+4])
		off += 4
		if off+int(nalLen) > len(avcc) {
			t.Fatalf("nal %d length %d overflows avcc (%d bytes left)", nalCount, nalLen, len(avcc)-off)
		}
		off += int(nalLen)
		nalCount++
	}
	if off != len(avcc) {
		t.Fatalf("parsed %d bytes of %d total avcc", off, len(avcc))
	}
	if nalCount == 0 {
		t.Fatal("expected at least one NAL unit")
	}
}
