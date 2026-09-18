// Package id membuat pengenal yang terurut menurut waktu.
package id

import (
	"crypto/rand"
	"encoding/base32"
	"time"
)

// Alfabet Crockford base32: tanpa i, l, o, u — supaya tidak tertukar saat dibaca
// manusia (misal saat menelusuri satu baris di log).
var enc = base32.
	NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").
	WithPadding(base32.NoPadding)

// New mengembalikan 26 karakter: 48 bit milidetik + 80 bit acak.
//
// Terurut waktu, jadi ORDER BY id bermakna dan sisipan baru selalu jatuh di
// ujung indeks B-tree — tidak seperti UUIDv4 yang menyebar acak.
func New() string {
	var b [16]byte
	ms := uint64(time.Now().UTC().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	if _, err := rand.Read(b[6:]); err != nil {
		panic("id: sumber acak sistem gagal: " + err.Error())
	}
	return enc.EncodeToString(b[:])
}
