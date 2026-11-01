package sfnt

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// fakeFont builds the smallest sfnt that has a maxp table.
func fakeFont(tag string, glyphs uint16) []byte {
	b := make([]byte, 12+16+6)
	copy(b, tag)
	binary.BigEndian.PutUint16(b[4:], 1) // numTables
	copy(b[12:], "maxp")
	binary.BigEndian.PutUint32(b[20:], 28) // offset
	binary.BigEndian.PutUint32(b[24:], 6)  // length
	binary.BigEndian.PutUint16(b[32:], glyphs)
	return b
}

func TestGlyphCount(t *testing.T) {
	dir := t.TempDir()
	for tag, want := range map[string]int{"\x00\x01\x00\x00": 9461, "OTTO": 12} {
		p := filepath.Join(dir, "f")
		os.WriteFile(p, fakeFont(tag, uint16(want)), 0o644)
		if got, err := GlyphCount(p); err != nil || got != want {
			t.Errorf("tag %q: got %d, %v; want %d", tag, got, err, want)
		}
	}
}

func TestGlyphCountReaderStopsAtMaxp(t *testing.T) {
	b := fakeFont("OTTO", 77)
	got, err := GlyphCountReader(bytes.NewReader(b[:len(b)])) // exactly up to the end of maxp
	if err != nil || got != 77 {
		t.Errorf("got %d, %v; want 77", got, err)
	}
	if _, err := GlyphCountReader(bytes.NewReader(b[:len(b)-1])); err == nil {
		t.Error("truncated maxp should be rejected")
	}
}

func TestGlyphCountRejectsNonFonts(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x")
	os.WriteFile(p, []byte("font-bytes"), 0o644)
	if _, err := GlyphCount(p); err == nil {
		t.Error("plain bytes should be rejected")
	}
	if _, err := GlyphCount(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("missing file should error")
	}
}
