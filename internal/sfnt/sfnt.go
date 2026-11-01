// Package sfnt reads the little we need from TrueType and OpenType files.
package sfnt

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

var errNotFont = errors.New("not a TrueType/OpenType font")

// GlyphCount returns the number of glyphs in the font file at path.
func GlyphCount(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return GlyphCountReader(f)
}

// GlyphCountReader reads the glyph count from the maxp table of a font
// streamed from its first byte. It reads only up to the end of that table, so
// it can stop early on a partly downloaded or decompressed stream. TrueType
// collections are not supported.
func GlyphCountReader(r io.Reader) (int, error) {
	var head [12]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return 0, errNotFont
	}
	switch string(head[:4]) {
	case "\x00\x01\x00\x00", "OTTO", "true":
	default:
		return 0, errNotFont
	}
	n := int(binary.BigEndian.Uint16(head[4:6]))
	var maxpOffset int64 = -1
	var rec [16]byte
	for i := 0; i < n; i++ {
		if _, err := io.ReadFull(r, rec[:]); err != nil {
			return 0, errNotFont
		}
		if string(rec[:4]) == "maxp" {
			maxpOffset = int64(binary.BigEndian.Uint32(rec[8:12]))
		}
	}
	pos := int64(12 + 16*n)
	if maxpOffset < pos {
		return 0, errNotFont
	}
	if _, err := io.CopyN(io.Discard, r, maxpOffset-pos); err != nil {
		return 0, errNotFont
	}
	var maxp [6]byte
	if _, err := io.ReadFull(r, maxp[:]); err != nil {
		return 0, errNotFont
	}
	return int(binary.BigEndian.Uint16(maxp[4:6])), nil
}
