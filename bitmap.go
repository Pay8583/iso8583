package iso8583

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// Bitmap represents the primary and optional secondary bitmap of an ISO 8583 message.
// Bits are numbered 1–128, where bit 1 is the most-significant bit of the first byte.
type Bitmap struct {
	Primary   uint64
	Secondary uint64
}

// ParseBitmap decodes a bitmap from bytes, returning the bitmap and bytes consumed
// (8 or 16 depending on whether the secondary bitmap is present).
func ParseBitmap(src []byte) (*Bitmap, int, error) {
	if len(src) < 8 {
		return nil, 0, fmt.Errorf("bitmap: need at least 8 bytes, got %d", len(src))
	}
	b := &Bitmap{
		Primary: binary.BigEndian.Uint64(src[:8]),
	}
	consumed := 8
	// Bit 1 of primary indicates secondary bitmap is present.
	if b.primaryBit(1) {
		if len(src) < 16 {
			return nil, 0, fmt.Errorf("bitmap: secondary bitmap indicated but only %d bytes available", len(src))
		}
		b.Secondary = binary.BigEndian.Uint64(src[8:16])
		consumed = 16
	}
	return b, consumed, nil
}

// IsSet reports whether the given field (1–128) is present.
func (b *Bitmap) IsSet(field int) bool {
	switch {
	case field < 1 || field > 128:
		return false
	case field <= 64:
		return b.primaryBit(field)
	default:
		return b.secondaryBit(field - 64)
	}
}

// Set marks a field (1–128) as present.
func (b *Bitmap) Set(field int) {
	switch {
	case field < 1 || field > 128:
		return
	case field <= 64:
		b.Primary |= 1 << (63 - (field - 1))
	default:
		if !b.primaryBit(1) {
			b.Primary |= 1 << 63 // set bit 1 (secondary bitmap indicator)
		}
		b.Secondary |= 1 << (63 - (field - 65))
	}
}

// Clear marks a field (1–128) as absent.
func (b *Bitmap) Clear(field int) {
	switch {
	case field < 1 || field > 128:
		return
	case field <= 64:
		b.Primary &^= 1 << (63 - (field - 1))
	default:
		b.Secondary &^= 1 << (63 - (field - 65))
	}
}

// HasSecondary reports whether the secondary bitmap is present (bit 1 is set).
func (b *Bitmap) HasSecondary() bool {
	return b.primaryBit(1)
}

// PresentFields returns the sorted list of field numbers (2–128) that are set.
// Field 1 (the bitmap indicator itself) is excluded.
func (b *Bitmap) PresentFields() []int {
	var fields []int
	for i := 2; i <= 64; i++ {
		if b.primaryBit(i) {
			fields = append(fields, i)
		}
	}
	if b.HasSecondary() {
		for i := 1; i <= 64; i++ {
			if b.secondaryBit(i) {
				fields = append(fields, 64+i)
			}
		}
	}
	sort.Ints(fields)
	return fields
}

// Bytes returns the wire-format bytes of the bitmap (8 or 16 bytes, big-endian).
func (b *Bitmap) Bytes() []byte {
	if b.HasSecondary() {
		out := make([]byte, 16)
		binary.BigEndian.PutUint64(out[:8], b.Primary)
		binary.BigEndian.PutUint64(out[8:], b.Secondary)
		return out
	}
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, b.Primary)
	return out
}

// Reset clears all bits.
func (b *Bitmap) Reset() {
	b.Primary = 0
	b.Secondary = 0
}

// primaryBit returns the value of bit n (1–64) within the primary bitmap.
func (b *Bitmap) primaryBit(n int) bool {
	return b.Primary&(1<<(63-(n-1))) != 0
}

// secondaryBit returns the value of bit n (1–64) within the secondary bitmap.
func (b *Bitmap) secondaryBit(n int) bool {
	return b.Secondary&(1<<(63-(n-1))) != 0
}
